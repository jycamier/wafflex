package cmd

import (
	"fmt"
	"log/slog"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jycamier/wafflex/internal/hash"
	"github.com/jycamier/wafflex/internal/tui"
	"github.com/spf13/cobra"
)

var exploreCmd = &cobra.Command{
	Use:   "explore [results.json]",
	Short: "Explore analysis results in an interactive TUI",
	Long:  `Opens an interactive terminal UI to browse and filter WAF analysis results. Without arguments, opens the most recent result for the current traffic source.`,
	Args:  cobra.MaximumNArgs(1),
	RunE:  runExplore,
}

func runExplore(cmd *cobra.Command, args []string) error {
	var resultsFile string

	if len(args) == 1 {
		resultsFile = args[0]
	} else {
		if appConfig == nil {
			return fmt.Errorf("config file required for auto-explore (or provide a file argument)")
		}
		trafficHash, err := hash.TrafficSourceHash(appConfig)
		if err != nil {
			return fmt.Errorf("failed to compute traffic hash: %w", err)
		}
		resultsDir := appConfig.ResultsDir
		if resultsDir == "" {
			resultsDir = "."
		}
		latest, err := hash.FindLatestResult(resultsDir, trafficHash)
		if err != nil {
			return fmt.Errorf("failed to find results: %w", err)
		}
		if latest == "" {
			return fmt.Errorf("no results found for current traffic source, run 'analyze' first")
		}
		resultsFile = latest
	}

	report, err := loadReport(resultsFile)
	if err != nil {
		return err
	}

	if len(report.Results) == 0 {
		slog.Info("no blocked requests found in the report")
		return nil
	}

	model := tui.NewExploreModel(report.Results)
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}
	return nil
}
