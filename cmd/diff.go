package cmd

import (
	"log/slog"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jycamier/wafflex/internal/baseline"
	"github.com/jycamier/wafflex/internal/diff"
	"github.com/jycamier/wafflex/internal/hash"
	"github.com/jycamier/wafflex/internal/tui"
	"github.com/spf13/cobra"
)

var diffCmd = &cobra.Command{
	Use:   "diff [results1.json] [results2.json]",
	Short: "Compare two analysis results",
	Long:  `Opens an interactive TUI to compare differences between two WAF analysis results. Without arguments, auto-detects the two most recent results for the current traffic source.`,
	Args:  cobra.MaximumNArgs(2),
	Run:   runDiff,
}

func runDiff(cmd *cobra.Command, args []string) {
	var file1, file2 string

	if len(args) == 2 {
		file1 = args[0]
		file2 = args[1]
	} else {
		resultsDir := resolveResultsDir()

		// file1 = baseline
		baselinePath, _ := baseline.GetPath(resultsDir)
		if baselinePath == "" {
			slog.Error("no baseline set (use 'wafflex baseline' to set one)")
			os.Exit(1)
		}
		file1 = baselinePath

		// file2 = latest result
		if appConfig == nil {
			slog.Error("config file required for auto-diff")
			os.Exit(1)
		}
		trafficHash, err := hash.TrafficSourceHash(appConfig)
		if err != nil {
			slog.Error("failed to compute traffic hash", "error", err)
			os.Exit(1)
		}
		latest, err := hash.FindLatestResult(resultsDir, trafficHash)
		if err != nil || latest == "" {
			slog.Error("no analysis results found, run 'analyze' first")
			os.Exit(1)
		}
		file2 = latest
	}

	report1 := loadReport(file1)
	report2 := loadReport(file2)

	diffReport := diff.Compare(report1, report2)
	if diffReport.Total() == 0 {
		slog.Info("no differences found")
		return
	}

	slog.Info("differences found",
		"total", diffReport.Total(),
		"added", len(diffReport.Added),
		"removed", len(diffReport.Removed),
		"modified", len(diffReport.Modified),
	)

	model := tui.NewDiffModel(diffReport)
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		slog.Error("TUI error", "error", err)
		os.Exit(1)
	}
}
