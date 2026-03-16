package cmd

import (
	"log/slog"
	"os"
	"path/filepath"

	"github.com/jycamier/wafflex/internal/baseline"
	"github.com/jycamier/wafflex/internal/hash"
	"github.com/spf13/cobra"
)

var baselineCmd = &cobra.Command{
	Use:   "baseline [- | set <file>]",
	Short: "Manage the baseline for diffs",
	Long:  `Without arguments, sets the most recent analysis as baseline. Use 'baseline set <file>' to choose a specific file, or 'baseline -' to swap with the previous baseline.`,
	Args:  cobra.MaximumNArgs(1),
	Run:   runBaseline,
}

var baselineSetCmd = &cobra.Command{
	Use:   "set <file>",
	Short: "Set a specific file as baseline",
	Args:  cobra.ExactArgs(1),
	Run:   runBaselineSet,
}

func init() {
	baselineCmd.AddCommand(baselineSetCmd)
}

func resolveResultsDir() string {
	if appConfig != nil && appConfig.ResultsDir != "" {
		return appConfig.ResultsDir
	}
	return "."
}

func runBaseline(cmd *cobra.Command, args []string) {
	// Handle "baseline -" as swap
	if len(args) == 1 && args[0] == "-" {
		runBaselineSwap()
		return
	}

	resultsDir := resolveResultsDir()

	if appConfig == nil {
		slog.Error("config file required to resolve latest analysis")
		os.Exit(1)
	}

	trafficHash, err := hash.TrafficSourceHash(appConfig)
	if err != nil {
		slog.Error("failed to compute traffic hash", "error", err)
		os.Exit(1)
	}

	latest, err := hash.FindLatestResult(resultsDir, trafficHash)
	if err != nil {
		slog.Error("failed to find latest result", "error", err)
		os.Exit(1)
	}
	if latest == "" {
		slog.Error("no analysis results found, run 'analyze' first")
		os.Exit(1)
	}

	filename := filepath.Base(latest)
	if err := baseline.Set(resultsDir, filename); err != nil {
		slog.Error("failed to set baseline", "error", err)
		os.Exit(1)
	}
	slog.Info("baseline set", "file", filename)
}

func runBaselineSet(cmd *cobra.Command, args []string) {
	resultsDir := resolveResultsDir()
	file := args[0]

	path := file
	if !filepath.IsAbs(file) {
		if _, err := os.Stat(file); err != nil {
			path = filepath.Join(resultsDir, file)
		}
	}
	if _, err := os.Stat(path); err != nil {
		slog.Error("file not found", "path", path)
		os.Exit(1)
	}

	filename := filepath.Base(path)
	if err := baseline.Set(resultsDir, filename); err != nil {
		slog.Error("failed to set baseline", "error", err)
		os.Exit(1)
	}
	slog.Info("baseline set", "file", filename)
}

func runBaselineSwap() {
	resultsDir := resolveResultsDir()

	if err := baseline.Swap(resultsDir); err != nil {
		slog.Error("failed to swap baseline", "error", err)
		os.Exit(1)
	}

	cur, _ := baseline.Get(resultsDir)
	slog.Info("baseline swapped", "current", cur)
}
