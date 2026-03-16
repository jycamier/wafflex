package cmd

import (
	"log/slog"
	"os"
	"sort"

	tea "github.com/charmbracelet/bubbletea"
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
		if appConfig == nil {
			slog.Error("config file required for auto-diff (or provide two file arguments)")
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

		results, err := hash.FindResults(resultsDir, trafficHash)
		if err != nil {
			slog.Error("failed to scan results", "error", err)
			os.Exit(1)
		}

		unique := uniqueByRulesHash(results)
		if len(unique) < 2 {
			slog.Error("need at least 2 results with different rules for auto-diff, provide files explicitly", "found", len(unique))
			os.Exit(1)
		}

		file1 = unique[1].Path // older
		file2 = unique[0].Path // newer
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

// uniqueByRulesHash returns results sorted by mtime desc, keeping only the
// most recent per unique rules hash.
func uniqueByRulesHash(results []hash.ResultFile) []hash.ResultFile {
	sort.Slice(results, func(i, j int) bool {
		fi, _ := os.Stat(results[i].Path)
		fj, _ := os.Stat(results[j].Path)
		if fi == nil || fj == nil {
			return false
		}
		return fi.ModTime().After(fj.ModTime())
	})

	seen := make(map[string]bool)
	var unique []hash.ResultFile
	for _, rf := range results {
		if !seen[rf.RulesHash] {
			seen[rf.RulesHash] = true
			unique = append(unique, rf)
		}
	}
	return unique
}
