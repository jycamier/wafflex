package cmd

import (
	"log/slog"
	"os"

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
	Run:   runExplore,
}

func runExplore(cmd *cobra.Command, args []string) {
	var resultsFile string

	if len(args) == 1 {
		resultsFile = args[0]
	} else {
		if appConfig == nil {
			slog.Error("config file required for auto-explore (or provide a file argument)")
			os.Exit(1)
		}
		trafficHash, err := hash.TrafficSourceHash(appConfig)
		if err != nil {
			slog.Error("failed to compute traffic hash", "error", err)
			os.Exit(1)
		}
		resultsDir := appConfig.ResultsDir
		if resultsDir == "" {
			resultsDir = "."
		}
		latest, err := hash.FindLatestResult(resultsDir, trafficHash)
		if err != nil {
			slog.Error("failed to find results", "error", err)
			os.Exit(1)
		}
		if latest == "" {
			slog.Error("no results found for current traffic source, run 'analyze' first")
			os.Exit(1)
		}
		resultsFile = latest
	}

	report := loadReport(resultsFile)

	if len(report.Results) == 0 {
		slog.Info("no blocked requests found in the report")
		return
	}

	model := tui.NewExploreModel(report.Results, report.Metadata.TotalRequests)
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		slog.Error("TUI error", "error", err)
		os.Exit(1)
	}
}
