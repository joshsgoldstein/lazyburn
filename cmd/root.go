package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/joshsgoldstein/lazyburn/internal/models"
	"github.com/joshsgoldstein/lazyburn/internal/output"
	"github.com/joshsgoldstein/lazyburn/internal/parser"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "lazyburn",
	Short: "Claude Code cost tracker",
	Long:  "Track Claude Code costs by folder, session, and date.\n\nFilter to a folder: lazyburn --path acme",
	RunE:  runRoot,
}

var (
	flagPath        string
	flagDepth       int
	flagAll         bool
	flagShowSession bool
	flagSince       string
	flagUntil       string
	flagExport      string
)

func init() {
	rootCmd.PersistentFlags().StringVar(&flagSince, "since", "", "Only include sessions after this date (YYYY-MM-DD)")
	rootCmd.PersistentFlags().StringVar(&flagUntil, "until", "", "Only include sessions before this date (YYYY-MM-DD)")

	rootCmd.Flags().StringVar(&flagPath, "path", "", "Filter by path substring (e.g. acme)")
	rootCmd.Flags().IntVar(&flagDepth, "depth", 2, "Folder depth to group by")
	rootCmd.Flags().BoolVar(&flagAll, "all", false, "Show all projects, ignore current directory")
	rootCmd.Flags().BoolVar(&flagShowSession, "sessions", false, "Also show per-session breakdown below folder summary")
	rootCmd.Flags().StringVar(&flagExport, "export", "", "Export results to CSV")

	rootCmd.AddCommand(sessionsCmd)
}

// Execute is the entry point called from main.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runRoot(cmd *cobra.Command, args []string) error {
	since, err := parseDate(flagSince)
	if err != nil {
		return fmt.Errorf("invalid --since: %w", err)
	}
	until, err := parseDate(flagUntil)
	if err != nil {
		return fmt.Errorf("invalid --until: %w", err)
	}

	home, _ := os.UserHomeDir()
	cwd, _ := os.Getwd()
	claudeDir := fmt.Sprintf("%s/.claude", home)

	var activeFilter string
	switch {
	case flagPath != "":
		activeFilter = flagPath
	case !flagAll && cwd != home:
		activeFilter = cwd
	}

	sessions, err := parser.ParseAllSessions(claudeDir, since, until, activeFilter)
	if err != nil {
		return err
	}
	if len(sessions) == 0 {
		label := activeFilter
		if label == "" {
			label = "anywhere"
		}
		fmt.Printf("No sessions found matching %s\n", label)
		return nil
	}

	if activeFilter != "" {
		fd := parser.FilterDepth(sessions, activeFilter, home)
		groupMap := parser.GroupByDepth(sessions, fd+1, home)
		groups := sortedGroups(groupMap)

		if len(groups) > 1 {
			output.PrintGroups(groups, sessions)
			if flagShowSession {
				fmt.Println()
				output.PrintSessions(sessions)
			}
			if flagExport != "" {
				if err := exportGroups(flagExport, groups); err != nil {
					return err
				}
				fmt.Printf("Exported to %s\n", flagExport)
			}
		} else {
			output.PrintSessions(sessions)
			if flagExport != "" {
				if err := exportSessions(flagExport, sessions); err != nil {
					return err
				}
				fmt.Printf("Exported to %s\n", flagExport)
			}
		}
	} else {
		groupMap := parser.GroupByDepth(sessions, flagDepth, home)
		groups := sortedGroups(groupMap)
		output.PrintGroups(groups, sessions)
		if flagShowSession {
			fmt.Println()
			output.PrintSessions(sessions)
		}
		if flagExport != "" {
			if err := exportGroups(flagExport, groups); err != nil {
				return err
			}
			fmt.Printf("Exported to %s\n", flagExport)
		}
	}
	return nil
}

func exportGroups(path string, groups []output.Group) error {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".json":
		return output.ExportGroupsJSON(path, groups)
	case ".md":
		return output.ExportGroupsMD(path, groups)
	default:
		return output.ExportGroupsCSV(path, groups)
	}
}

func exportSessions(path string, sessions []models.Session) error {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".json":
		return output.ExportSessionsJSON(path, sessions)
	case ".md":
		return output.ExportSessionsMD(path, sessions)
	default:
		return output.ExportSessionsCSV(path, sessions)
	}
}

func sortedGroups(groupMap map[string][]models.Session) []output.Group {
	groups := make([]output.Group, 0, len(groupMap))
	for folder, sessions := range groupMap {
		groups = append(groups, output.Group{Folder: folder, Sessions: sessions})
	}
	sort.Slice(groups, func(i, j int) bool {
		return parser.Aggregate(groups[i].Sessions).Cost > parser.Aggregate(groups[j].Sessions).Cost
	})
	return groups
}

func parseDate(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}
	return time.Parse("2006-01-02", s)
}
