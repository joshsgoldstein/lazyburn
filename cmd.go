package main

import (
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/spf13/cobra"
)

func parseDate(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}
	return time.Parse("2006-01-02", s)
}

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

	sessions, err := ParseAllSessions(claudeDir, since, until, activeFilter)
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
		fd := FilterDepth(sessions, activeFilter, home)
		groupMap := GroupByDepth(sessions, fd+1, home)
		groups := sortedGroups(groupMap)

		if len(groups) > 1 {
			PrintGroups(groups, sessions)
			if flagShowSession {
				fmt.Println()
				PrintSessions(sessions)
			}
			if flagExport != "" {
				if err := ExportGroupsCSV(flagExport, groups); err != nil {
					return err
				}
				fmt.Printf("Exported to %s\n", flagExport)
			}
		} else {
			PrintSessions(sessions)
			if flagExport != "" {
				if err := ExportSessionsCSV(flagExport, sessions); err != nil {
					return err
				}
				fmt.Printf("Exported to %s\n", flagExport)
			}
		}
	} else {
		groupMap := GroupByDepth(sessions, flagDepth, home)
		groups := sortedGroups(groupMap)
		PrintGroups(groups, sessions)
		if flagShowSession {
			fmt.Println()
			PrintSessions(sessions)
		}
		if flagExport != "" {
			if err := ExportGroupsCSV(flagExport, groups); err != nil {
				return err
			}
			fmt.Printf("Exported to %s\n", flagExport)
		}
	}
	return nil
}

// sessionsCmd is the `lazyburn sessions` subcommand.
var sessionsCmd = &cobra.Command{
	Use:   "sessions",
	Short: "Show individual session breakdown",
	Long: `Show individual session breakdown.

  lazyburn sessions               # current directory
  lazyburn sessions --path acme   # filter by path`,
	RunE: runSessions,
}

var (
	flagSessionsPath   string
	flagSessionsExport string
)

func init() {
	sessionsCmd.Flags().StringVar(&flagSessionsPath, "path", "", "Filter by path substring")
	sessionsCmd.Flags().StringVar(&flagSessionsExport, "export", "", "Export results to CSV")
}

func runSessions(cmd *cobra.Command, args []string) error {
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

	pathFilter := flagSessionsPath
	if pathFilter == "" && cwd != home {
		pathFilter = cwd
	}

	sessions, err := ParseAllSessions(claudeDir, since, until, pathFilter)
	if err != nil {
		return err
	}
	if len(sessions) == 0 {
		fmt.Println("No sessions found.")
		return nil
	}

	PrintSessions(sessions)

	if flagSessionsExport != "" {
		if err := ExportSessionsCSV(flagSessionsExport, sessions); err != nil {
			return err
		}
		fmt.Printf("Exported to %s\n", flagSessionsExport)
	}
	return nil
}

// sortedGroups converts the map from GroupByDepth into a slice sorted by cost descending.
func sortedGroups(groupMap map[string][]Session) []Group {
	groups := make([]Group, 0, len(groupMap))
	for folder, sessions := range groupMap {
		groups = append(groups, Group{Folder: folder, Sessions: sessions})
	}
	sort.Slice(groups, func(i, j int) bool {
		return Aggregate(groups[i].Sessions).Cost > Aggregate(groups[j].Sessions).Cost
	})
	return groups
}
