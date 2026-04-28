package cmd

import (
	"fmt"
	"os"

	"github.com/joshsgoldstein/lazyburn/internal/output"
	"github.com/joshsgoldstein/lazyburn/internal/parser"
	"github.com/spf13/cobra"
)

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

	sessions, err := parser.ParseAllSessions(claudeDir, since, until, pathFilter)
	if err != nil {
		return err
	}
	if len(sessions) == 0 {
		fmt.Println("No sessions found.")
		return nil
	}

	output.PrintSessions(sessions)

	if flagSessionsExport != "" {
		if err := exportSessions(flagSessionsExport, sessions); err != nil {
			return err
		}
		fmt.Printf("Exported to %s\n", flagSessionsExport)
	}
	return nil
}
