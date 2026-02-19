package cmd

import (
	"fmt"
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
	RunE:  runDiff,
}

func runDiff(cmd *cobra.Command, args []string) error {
	var file1, file2 string

	if len(args) == 2 {
		file1 = args[0]
		file2 = args[1]
	} else {
		if appConfig == nil {
			return fmt.Errorf("config file required for auto-diff (or provide two file arguments)")
		}
		trafficHash, err := hash.TrafficSourceHash(appConfig)
		if err != nil {
			return fmt.Errorf("failed to compute traffic hash: %w", err)
		}

		resultsDir := appConfig.ResultsDir
		if resultsDir == "" {
			resultsDir = "."
		}

		results, err := hash.FindResults(resultsDir, trafficHash)
		if err != nil {
			return fmt.Errorf("failed to scan results: %w", err)
		}

		unique := uniqueByRulesHash(results)
		if len(unique) < 2 {
			return fmt.Errorf("need at least 2 results with different rules for auto-diff (found %d), provide files explicitly", len(unique))
		}

		file1 = unique[1].Path // older
		file2 = unique[0].Path // newer
	}

	report1, err := loadReport(file1)
	if err != nil {
		return err
	}
	report2, err := loadReport(file2)
	if err != nil {
		return err
	}

	diffReport := diff.Compare(report1, report2)
	if diffReport.Total() == 0 {
		slog.Info("no differences found")
		return nil
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
		return fmt.Errorf("TUI error: %w", err)
	}
	return nil
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
