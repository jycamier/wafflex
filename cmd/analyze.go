package cmd

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jycamier/wafflex/internal/diff"
	"github.com/jycamier/wafflex/internal/hash"
	"github.com/jycamier/wafflex/internal/models"
	"github.com/jycamier/wafflex/internal/tui"
	"github.com/jycamier/wafflex/internal/waf"
	"github.com/spf13/cobra"
)

var analyzeCmd = &cobra.Command{
	Use:   "analyze",
	Short: "Analyze traffic through Coraza WAF",
	Long:  `Processes traffic through Coraza WAF, saves results, and opens the explore TUI.`,
	Run:   runAnalyze,
}

func init() {
	analyzeCmd.Flags().StringP("gor-file", "g", "", "Path to traffic file (.gor or .custom)")
	analyzeCmd.Flags().StringP("coraza-config", "c", "", "Path to Coraza configuration file")
	analyzeCmd.Flags().StringP("output", "o", "", "Output JSON file path (explicit path, skips TUI)")
	analyzeCmd.Flags().Bool("diff", false, "After analysis, diff with previous result (same traffic, different rules)")
}

func runAnalyze(cmd *cobra.Command, args []string) {
	gorFile, _ := cmd.Flags().GetString("gor-file")
	corazaConfig, _ := cmd.Flags().GetString("coraza-config")
	outputFile, _ := cmd.Flags().GetString("output")
	doDiff, _ := cmd.Flags().GetBool("diff")

	// Resolve from config if flags not set
	if gorFile == "" && appConfig != nil {
		gorFile = appConfig.Traffic.File
	}
	if corazaConfig == "" && appConfig != nil {
		corazaConfig = appConfig.CorazaConfig
	}

	if corazaConfig == "" {
		slog.Error("coraza config is required (use --coraza-config flag or config file)")
		os.Exit(1)
	}
	if _, err := os.Stat(corazaConfig); os.IsNotExist(err) {
		slog.Error("coraza config file does not exist", "path", corazaConfig)
		os.Exit(1)
	}

	explicitOutput := outputFile != ""

	// Compute hashes and resolve output path (when no explicit -o)
	var trafficHash, rulesHash string
	if !explicitOutput && appConfig != nil {
		var err error
		trafficHash, err = hash.TrafficSourceHash(appConfig)
		if err != nil {
			slog.Error("failed to compute traffic hash", "error", err)
			os.Exit(1)
		}
		rulesHash, err = hash.RulesHash(corazaConfig)
		if err != nil {
			slog.Error("failed to compute rules hash", "error", err)
			os.Exit(1)
		}

		resultsDir := appConfig.ResultsDir
		if resultsDir == "" {
			resultsDir = "."
		}
		if err := os.MkdirAll(resultsDir, 0755); err != nil {
			slog.Error("failed to create results directory", "error", err)
			os.Exit(1)
		}
		outputFile = filepath.Join(resultsDir, hash.ResultFileName(trafficHash, rulesHash))
	}

	if outputFile == "" {
		outputFile = "results.json"
	}

	// Skip analysis if result already exists
	if _, err := os.Stat(outputFile); err == nil {
		slog.Info("results already exist, skipping analysis", "path", outputFile)
	} else {
		executeAnalysis(gorFile, corazaConfig, outputFile, trafficHash, rulesHash)
	}

	// If explicit -o flag, no TUI (backward compat)
	if explicitOutput {
		return
	}

	// Load results for TUI
	report := loadReport(outputFile)

	if doDiff {
		launchDiffTUI(report, outputFile, trafficHash, rulesHash)
		return
	}
	launchExploreTUI(report)
}

func executeAnalysis(gorFile, corazaConfig, outputFile, trafficHash, rulesHash string) {
	engine, err := waf.NewWAFEngine(corazaConfig)
	if err != nil {
		slog.Error("failed to initialize WAF", "error", err)
		os.Exit(1)
	}
	slog.Info("WAF engine initialized", "engine", engine.Name(), "version", engine.Version())

	reader, cleanup := openTrafficReader(gorFile)
	defer cleanup()
	defer reader.Close()

	requests, errors := reader.ReadRequests(1000)
	resultsChan := make(chan *models.Result, 100)
	done := make(chan bool)

	var results []models.Result
	totalRequests := 0
	blockedRequests := 0

	go func() {
		for result := range resultsChan {
			if result.Blocked {
				blockedRequests++
				results = append(results, *result)
			}
		}
		done <- true
	}()

	workers := 4
	workChan := make(chan *http.Request, workers*2)
	workersDone := make(chan bool, workers)

	for i := 0; i < workers; i++ {
		go func() {
			for req := range workChan {
				result, err := engine.ProcessRequest(req)
				if err != nil {
					slog.Error("failed to process request", "error", err)
					continue
				}
				resultsChan <- result
			}
			workersDone <- true
		}()
	}

	for req := range requests {
		totalRequests++
		workChan <- req
		if totalRequests%100 == 0 {
			slog.Info("processing", "requests", totalRequests, "blocked", blockedRequests)
		}
	}

	close(workChan)
	for i := 0; i < workers; i++ {
		<-workersDone
	}
	close(resultsChan)
	<-done

	if err := <-errors; err != nil {
		slog.Warn("traffic read error", "error", err)
	}

	slog.Info("analysis complete", "total", totalRequests, "blocked", blockedRequests)

	report := models.AnalysisReport{
		Metadata: models.Metadata{
			Timestamp:       time.Now(),
			TotalRequests:   totalRequests,
			BlockedRequests: blockedRequests,
			TrafficHash:     trafficHash,
			RulesHash:       rulesHash,
		},
		Results: results,
	}

	file, err := os.Create(outputFile)
	if err != nil {
		slog.Error("failed to create output file", "error", err)
		os.Exit(1)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(report); err != nil {
		slog.Error("failed to write JSON", "error", err)
		os.Exit(1)
	}

	slog.Info("results saved", "path", outputFile)
}

func loadReport(path string) *models.AnalysisReport {
	f, err := os.Open(path)
	if err != nil {
		slog.Error("failed to open results", "error", err)
		os.Exit(1)
	}
	defer f.Close()

	var report models.AnalysisReport
	if err := json.NewDecoder(f).Decode(&report); err != nil {
		slog.Error("failed to parse results", "error", err)
		os.Exit(1)
	}
	return &report
}

func launchExploreTUI(report *models.AnalysisReport) {
	if len(report.Results) == 0 {
		slog.Info("no blocked requests found")
		return
	}
	model := tui.NewExploreModel(report.Results)
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		slog.Error("TUI error", "error", err)
		os.Exit(1)
	}
}

func launchDiffTUI(report *models.AnalysisReport, currentFile, trafficHash, rulesHash string) {
	resultsDir := filepath.Dir(currentFile)
	prevFile, err := hash.FindPreviousDiff(resultsDir, trafficHash, rulesHash)
	if err != nil {
		slog.Error("failed to find previous result", "error", err)
		os.Exit(1)
	}
	if prevFile == "" {
		slog.Warn("no previous result found for diff, falling back to explore")
		launchExploreTUI(report)
		return
	}

	prevReport := loadReport(prevFile)

	diffReport := diff.Compare(prevReport, report)
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
