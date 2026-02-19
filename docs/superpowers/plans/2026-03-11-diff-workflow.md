# Diff Workflow Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Hash-based result naming with auto-explore after analyze, auto-diff support, and skip-if-exists optimization.

**Architecture:** New `internal/hash` package computes traffic source and rules hashes. `cmd/analyze.go` uses hashes for file naming, skips analysis when result exists, and launches explore/diff TUI. `cmd/diff.go` and `cmd/explore.go` gain zero-arg auto-resolution modes.

**Tech Stack:** Go, SHA256, Cobra, Bubble Tea

**Spec:** `docs/superpowers/specs/2026-03-11-diff-workflow-design.md`

---

## File Structure

| Action | File | Responsibility |
|--------|------|----------------|
| Create | `internal/hash/hash.go` | `TrafficSourceHash`, `RulesHash`, `ResultFileName` |
| Create | `internal/hash/hash_test.go` | Unit tests for hash functions |
| Create | `internal/hash/results.go` | `FindResults`, `FindPreviousDiff` — scan results dir |
| Create | `internal/hash/results_test.go` | Unit tests for result file scanning |
| Modify | `internal/models/result.go` | Add `TrafficHash`, `RulesHash` to `Metadata` |
| Modify | `cmd/analyze.go` | Hash-based naming, skip-if-exists, auto-explore, `--diff` flag |
| Modify | `cmd/diff.go` | Zero-arg auto-diff mode |
| Modify | `cmd/explore.go` | Zero-arg auto-explore mode |

---

## Chunk 1: Hash Package

### Task 1: Create `internal/hash/hash.go` with hash functions

**Files:**
- Create: `internal/hash/hash.go`
- Create: `internal/hash/hash_test.go`

- [ ] **Step 1: Write failing tests for `TrafficSourceHash`**

```go
// internal/hash/hash_test.go
package hash

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jycamier/wafflex/internal/config"
)

func TestTrafficSourceHashParquet(t *testing.T) {
	cfg := &config.Config{
		Traffic: config.TrafficConfig{
			Type:  "parquet",
			Query: "SELECT * FROM read_parquet('s3://bucket/*.parquet')",
		},
	}
	h, err := TrafficSourceHash(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(h) != 12 {
		t.Errorf("hash length = %d, want 12", len(h))
	}

	// Same query = same hash
	h2, _ := TrafficSourceHash(cfg)
	if h != h2 {
		t.Errorf("same config produced different hashes: %s vs %s", h, h2)
	}
}

func TestTrafficSourceHashParquetDifferentQuery(t *testing.T) {
	cfg1 := &config.Config{Traffic: config.TrafficConfig{Type: "parquet", Query: "SELECT 1"}}
	cfg2 := &config.Config{Traffic: config.TrafficConfig{Type: "parquet", Query: "SELECT 2"}}

	h1, _ := TrafficSourceHash(cfg1)
	h2, _ := TrafficSourceHash(cfg2)
	if h1 == h2 {
		t.Error("different queries should produce different hashes")
	}
}

func TestTrafficSourceHashGor(t *testing.T) {
	dir := t.TempDir()
	gorFile := filepath.Join(dir, "traffic.gor")
	os.WriteFile(gorFile, []byte("GET /test HTTP/1.1\r\nHost: example.com\r\n\r\n"), 0644)

	cfg := &config.Config{
		Traffic: config.TrafficConfig{
			Type: "gor",
			File: gorFile,
		},
	}
	h, err := TrafficSourceHash(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(h) != 12 {
		t.Errorf("hash length = %d, want 12", len(h))
	}
}

func TestTrafficSourceHashGorFileNotFound(t *testing.T) {
	cfg := &config.Config{
		Traffic: config.TrafficConfig{
			Type: "gor",
			File: "/nonexistent/traffic.gor",
		},
	}
	_, err := TrafficSourceHash(cfg)
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestRulesHash(t *testing.T) {
	dir := t.TempDir()
	confFile := filepath.Join(dir, "rules.conf")
	os.WriteFile(confFile, []byte("SecRuleEngine On\n"), 0644)

	h, err := RulesHash(confFile)
	if err != nil {
		t.Fatal(err)
	}
	if len(h) != 12 {
		t.Errorf("hash length = %d, want 12", len(h))
	}
}

func TestRulesHashDifferentContent(t *testing.T) {
	dir := t.TempDir()
	f1 := filepath.Join(dir, "rules1.conf")
	f2 := filepath.Join(dir, "rules2.conf")
	os.WriteFile(f1, []byte("SecRuleEngine On\n"), 0644)
	os.WriteFile(f2, []byte("SecRuleEngine Off\n"), 0644)

	h1, _ := RulesHash(f1)
	h2, _ := RulesHash(f2)
	if h1 == h2 {
		t.Error("different rules should produce different hashes")
	}
}

func TestResultFileName(t *testing.T) {
	name := ResultFileName("cb4004b7dd82", "a1b2c3d4e5f6")
	if name != "cb4004b7dd82-a1b2c3d4e5f6.json" {
		t.Errorf("got %q, want %q", name, "cb4004b7dd82-a1b2c3d4e5f6.json")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/hash/ -v`
Expected: compilation error (package doesn't exist)

- [ ] **Step 3: Implement `internal/hash/hash.go`**

```go
// internal/hash/hash.go
package hash

import (
	"crypto/sha256"
	"fmt"
	"os"

	"github.com/jycamier/wafflex/internal/config"
)

// TrafficSourceHash returns a 12-char hex hash identifying the traffic source.
// Parquet: hash of the SQL query. GOR/Custom: hash of the file content.
func TrafficSourceHash(cfg *config.Config) (string, error) {
	switch cfg.Traffic.Type {
	case "parquet":
		if cfg.Traffic.Query == "" {
			return "", fmt.Errorf("parquet query is required for traffic hash")
		}
		return shortHash([]byte(cfg.Traffic.Query)), nil
	default:
		if cfg.Traffic.File == "" {
			return "", fmt.Errorf("traffic file is required for traffic hash")
		}
		data, err := os.ReadFile(cfg.Traffic.File)
		if err != nil {
			return "", fmt.Errorf("failed to read traffic file: %w", err)
		}
		return shortHash(data), nil
	}
}

// RulesHash returns a 12-char hex hash of the Coraza config file content.
func RulesHash(corazaConfigPath string) (string, error) {
	data, err := os.ReadFile(corazaConfigPath)
	if err != nil {
		return "", fmt.Errorf("failed to read coraza config: %w", err)
	}
	return shortHash(data), nil
}

// ResultFileName returns the hash-based result file name.
func ResultFileName(trafficHash, rulesHash string) string {
	return trafficHash + "-" + rulesHash + ".json"
}

func shortHash(data []byte) string {
	h := sha256.Sum256(data)
	return fmt.Sprintf("%x", h)[:12]
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/hash/ -v`
Expected: all PASS

- [ ] **Step 5: Commit**

```bash
git add internal/hash/hash.go internal/hash/hash_test.go
git commit -m "feat(hash): add traffic source and rules hash computation"
```

---

### Task 2: Create result file scanning functions

**Files:**
- Create: `internal/hash/results.go`
- Create: `internal/hash/results_test.go`

- [ ] **Step 1: Write failing tests**

```go
// internal/hash/results_test.go
package hash

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFindResultsMatchesPrefix(t *testing.T) {
	dir := t.TempDir()

	// Create matching files
	os.WriteFile(filepath.Join(dir, "aabbccddee11-111111111111.json"), []byte("{}"), 0644)
	os.WriteFile(filepath.Join(dir, "aabbccddee11-222222222222.json"), []byte("{}"), 0644)
	// Create non-matching file
	os.WriteFile(filepath.Join(dir, "xxxxyyyyzzzz-333333333333.json"), []byte("{}"), 0644)

	results, err := FindResults(dir, "aabbccddee11")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Errorf("got %d results, want 2", len(results))
	}
}

func TestFindResultsEmptyDir(t *testing.T) {
	dir := t.TempDir()
	results, err := FindResults(dir, "aabbccddee11")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 0 {
		t.Errorf("got %d results, want 0", len(results))
	}
}

func TestFindPreviousDiff(t *testing.T) {
	dir := t.TempDir()

	// Create two files with same traffic hash but different rules
	f1 := filepath.Join(dir, "aabbccddee11-111111111111.json")
	f2 := filepath.Join(dir, "aabbccddee11-222222222222.json")
	os.WriteFile(f1, []byte("{}"), 0644)
	time.Sleep(10 * time.Millisecond) // ensure different mtime
	os.WriteFile(f2, []byte("{}"), 0644)

	prev, err := FindPreviousDiff(dir, "aabbccddee11", "222222222222")
	if err != nil {
		t.Fatal(err)
	}
	if prev == "" {
		t.Fatal("expected to find previous result")
	}
	if filepath.Base(prev) != "aabbccddee11-111111111111.json" {
		t.Errorf("got %q, want aabbccddee11-111111111111.json", filepath.Base(prev))
	}
}

func TestFindPreviousDiffNoPrevious(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "aabbccddee11-111111111111.json"), []byte("{}"), 0644)

	prev, err := FindPreviousDiff(dir, "aabbccddee11", "111111111111")
	if err != nil {
		t.Fatal(err)
	}
	if prev != "" {
		t.Errorf("expected empty, got %q", prev)
	}
}

func TestFindLatestResult(t *testing.T) {
	dir := t.TempDir()
	f1 := filepath.Join(dir, "aabbccddee11-111111111111.json")
	os.WriteFile(f1, []byte("{}"), 0644)
	time.Sleep(10 * time.Millisecond)
	f2 := filepath.Join(dir, "aabbccddee11-222222222222.json")
	os.WriteFile(f2, []byte("{}"), 0644)

	latest, err := FindLatestResult(dir, "aabbccddee11")
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(latest) != "aabbccddee11-222222222222.json" {
		t.Errorf("got %q, want aabbccddee11-222222222222.json", filepath.Base(latest))
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/hash/ -run "FindResults|FindPrevious|FindLatest" -v`
Expected: compilation error

- [ ] **Step 3: Implement `internal/hash/results.go`**

```go
// internal/hash/results.go
package hash

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ResultFile represents a result file parsed from its hash-based name.
type ResultFile struct {
	Path        string
	TrafficHash string
	RulesHash   string
}

// FindResults returns all result files in dir matching the given traffic hash prefix.
func FindResults(dir, trafficHash string) ([]ResultFile, error) {
	pattern := filepath.Join(dir, trafficHash+"-*.json")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}

	var results []ResultFile
	for _, path := range matches {
		rf, ok := ParseResultFileName(filepath.Base(path))
		if ok {
			rf.Path = path
			results = append(results, rf)
		}
	}
	return results, nil
}

// FindLatestResult returns the most recent result file for a traffic hash.
func FindLatestResult(dir, trafficHash string) (string, error) {
	results, err := FindResults(dir, trafficHash)
	if err != nil {
		return "", err
	}
	if len(results) == 0 {
		return "", nil
	}
	return newestByMtime(results)
}

// FindPreviousDiff finds the most recent result with the same traffic hash
// but a different rules hash. Returns empty string if none found.
func FindPreviousDiff(dir, trafficHash, currentRulesHash string) (string, error) {
	results, err := FindResults(dir, trafficHash)
	if err != nil {
		return "", err
	}

	var candidates []ResultFile
	for _, rf := range results {
		if rf.RulesHash != currentRulesHash {
			candidates = append(candidates, rf)
		}
	}

	if len(candidates) == 0 {
		return "", nil
	}
	return newestByMtime(candidates)
}

// ParseResultFileName extracts hashes from a "<traffic>-<rules>.json" filename.
func ParseResultFileName(name string) (ResultFile, bool) {
	name = strings.TrimSuffix(name, ".json")
	parts := strings.SplitN(name, "-", 2)
	if len(parts) != 2 || len(parts[0]) != 12 || len(parts[1]) != 12 {
		return ResultFile{}, false
	}
	return ResultFile{TrafficHash: parts[0], RulesHash: parts[1]}, true
}

func newestByMtime(files []ResultFile) (string, error) {
	sort.Slice(files, func(i, j int) bool {
		fi, _ := os.Stat(files[i].Path)
		fj, _ := os.Stat(files[j].Path)
		if fi == nil || fj == nil {
			return false
		}
		return fi.ModTime().After(fj.ModTime())
	})
	return files[0].Path, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/hash/ -v`
Expected: all PASS

- [ ] **Step 5: Commit**

```bash
git add internal/hash/results.go internal/hash/results_test.go
git commit -m "feat(hash): add result file scanning and matching functions"
```

---

## Chunk 2: Metadata and Command Updates

### Task 3: Add hashes to `Metadata` model

**Files:**
- Modify: `internal/models/result.go`

- [ ] **Step 1: Add `TrafficHash` and `RulesHash` fields to `Metadata`**

In `internal/models/result.go`, add two fields to the `Metadata` struct:

```go
type Metadata struct {
	Timestamp       time.Time `json:"timestamp"`
	TotalRequests   int       `json:"total_requests"`
	BlockedRequests int       `json:"blocked_requests"`
	TrafficHash     string    `json:"traffic_hash,omitempty"`
	RulesHash       string    `json:"rules_hash,omitempty"`
}
```

- [ ] **Step 2: Build to verify compilation**

Run: `go build ./...`
Expected: success

- [ ] **Step 3: Commit**

```bash
git add internal/models/result.go
git commit -m "feat(models): add traffic and rules hashes to metadata"
```

---

### Task 4: Rewrite `cmd/analyze.go` with hash-based naming, skip-if-exists, and auto-explore

**Files:**
- Modify: `cmd/analyze.go`

- [ ] **Step 1: Add `--diff` flag in `init()`**

```go
func init() {
	analyzeCmd.Flags().StringP("gor-file", "g", "", "Path to traffic file (.gor or .custom)")
	analyzeCmd.Flags().StringP("coraza-config", "c", "", "Path to Coraza configuration file")
	analyzeCmd.Flags().StringP("output", "o", "", "Output JSON file path (explicit path, skips TUI)")
	analyzeCmd.Flags().Bool("diff", false, "After analysis, diff with previous result (same traffic, different rules)")
}
```

- [ ] **Step 2: Rewrite `runAnalyze` with hash logic, skip-if-exists, and TUI launch**

Replace the full `runAnalyze` function. Key changes:
1. Compute `trafficHash` and `rulesHash` early
2. If `-o` is not set, derive output path from hashes
3. If output file already exists, skip analysis and log
4. After write (or skip), launch explore TUI or diff TUI based on `--diff`
5. If `--diff` but no previous result found, warn and fall back to explore

```go
func runAnalyze(cmd *cobra.Command, args []string) error {
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
		return fmt.Errorf("coraza config is required (use --coraza-config flag or config file)")
	}
	if _, err := os.Stat(corazaConfig); os.IsNotExist(err) {
		return fmt.Errorf("coraza config file does not exist: %s", corazaConfig)
	}

	explicitOutput := outputFile != ""

	// Compute hashes and resolve output path (when no explicit -o)
	var trafficHash, rulesHash string
	if !explicitOutput && appConfig != nil {
		var err error
		trafficHash, err = hash.TrafficSourceHash(appConfig)
		if err != nil {
			return fmt.Errorf("failed to compute traffic hash: %w", err)
		}
		rulesHash, err = hash.RulesHash(corazaConfig)
		if err != nil {
			return fmt.Errorf("failed to compute rules hash: %w", err)
		}

		resultsDir := appConfig.ResultsDir
		if resultsDir == "" {
			resultsDir = "."
		}
		if err := os.MkdirAll(resultsDir, 0755); err != nil {
			return fmt.Errorf("failed to create results directory: %w", err)
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
		// Run analysis
		if err := executeAnalysis(gorFile, corazaConfig, outputFile, trafficHash, rulesHash); err != nil {
			return err
		}
	}

	// If explicit -o flag, no TUI (backward compat)
	if explicitOutput {
		return nil
	}

	// Load results for TUI
	report, err := loadReport(outputFile)
	if err != nil {
		return err
	}

	// Launch diff or explore TUI
	if doDiff {
		return launchDiffTUI(report, outputFile, trafficHash, rulesHash)
	}
	return launchExploreTUI(report)
}

func executeAnalysis(gorFile, corazaConfig, outputFile, trafficHash, rulesHash string) error {
	engine, err := waf.NewWAFEngine(corazaConfig)
	if err != nil {
		return fmt.Errorf("failed to initialize WAF: %w", err)
	}
	slog.Info("WAF engine initialized", "engine", engine.Name(), "version", engine.Version())

	reader, cleanup, err := openTrafficReader(gorFile)
	if err != nil {
		return fmt.Errorf("failed to open traffic source: %w", err)
	}
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
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(report); err != nil {
		return fmt.Errorf("failed to write JSON: %w", err)
	}

	slog.Info("results saved", "path", outputFile)
	return nil
}

func loadReport(path string) (*models.AnalysisReport, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open results: %w", err)
	}
	defer f.Close()

	var report models.AnalysisReport
	if err := json.NewDecoder(f).Decode(&report); err != nil {
		return nil, fmt.Errorf("failed to parse results: %w", err)
	}
	return &report, nil
}

func launchExploreTUI(report *models.AnalysisReport) error {
	if len(report.Results) == 0 {
		slog.Info("no blocked requests found")
		return nil
	}
	model := tui.NewExploreModel(report.Results)
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}
	return nil
}

func launchDiffTUI(report *models.AnalysisReport, currentFile, trafficHash, rulesHash string) error {
	resultsDir := filepath.Dir(currentFile)
	prevFile, err := hash.FindPreviousDiff(resultsDir, trafficHash, rulesHash)
	if err != nil {
		return fmt.Errorf("failed to find previous result: %w", err)
	}
	if prevFile == "" {
		slog.Warn("no previous result found for diff, falling back to explore")
		return launchExploreTUI(report)
	}

	prevReport, err := loadReport(prevFile)
	if err != nil {
		return err
	}

	diffReport := diff.Compare(prevReport, report)
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
```

- [ ] **Step 3: Add missing imports**

Update the import block at the top of `cmd/analyze.go`:

```go
import (
	"encoding/json"
	"fmt"
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
```

- [ ] **Step 4: Build to verify compilation**

Run: `go build ./...`
Expected: success

- [ ] **Step 5: Commit**

```bash
git add cmd/analyze.go
git commit -m "feat(analyze): hash-based naming, skip-if-exists, auto-explore and --diff"
```

---

### Task 5: Update `cmd/diff.go` for zero-arg auto-diff

**Files:**
- Modify: `cmd/diff.go`

- [ ] **Step 1: Change `Args` from `ExactArgs(2)` to `MaximumNArgs(2)` and update `runDiff`**

```go
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
		// Explicit mode
		file1 = args[0]
		file2 = args[1]
	} else {
		// Auto-diff: find two most recent results for current traffic source
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

		// Need at least 2 results with different rules hashes
		unique := uniqueByRulesHash(results)
		if len(unique) < 2 {
			return fmt.Errorf("need at least 2 results with different rules for auto-diff (found %d), provide files explicitly", len(unique))
		}

		// Take two most recent (by mtime) with different rules hashes
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
	// Sort by mtime descending
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
```

- [ ] **Step 2: Update imports**

```go
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
```

- [ ] **Step 3: Build to verify compilation**

Run: `go build ./...`
Expected: success

- [ ] **Step 4: Commit**

```bash
git add cmd/diff.go
git commit -m "feat(diff): add zero-arg auto-diff using traffic hash from config"
```

---

### Task 6: Update `cmd/explore.go` for zero-arg mode

**Files:**
- Modify: `cmd/explore.go`

- [ ] **Step 1: Change `Args` to `MaximumNArgs(1)` and update `runExplore`**

```go
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
		// Auto-resolve: find most recent result for current traffic source
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
```

- [ ] **Step 2: Update imports**

```go
import (
	"fmt"
	"log/slog"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jycamier/wafflex/internal/hash"
	"github.com/jycamier/wafflex/internal/models"
	"github.com/jycamier/wafflex/internal/tui"
	"github.com/spf13/cobra"
)
```

Remove `encoding/json` and `os` imports (now handled by shared `loadReport` in analyze.go).

- [ ] **Step 3: Move `loadReport` to a shared location**

Note: `loadReport` is defined in `cmd/analyze.go` and used by both `cmd/diff.go` and `cmd/explore.go`. Since they are all in package `cmd`, it is already shared. No move needed.

- [ ] **Step 4: Build and run all tests**

Run: `go build ./... && go test ./...`
Expected: all pass

- [ ] **Step 5: Commit**

```bash
git add cmd/explore.go
git commit -m "feat(explore): add zero-arg auto-explore using traffic hash from config"
```

---

### Task 7: Final build, test, and cleanup

- [ ] **Step 1: Run full test suite**

Run: `go test ./... -v`
Expected: all PASS

- [ ] **Step 2: Manual smoke test**

```bash
# Test hash-based naming
wafflex analyze -c coraza-test.conf
# → should create results/<hash>-<hash>.json and open explore TUI

# Test skip-if-exists (run again)
wafflex analyze
# → should log "results already exist, skipping analysis" and open TUI

# Test --diff (modify rules first, then analyze)
wafflex analyze --diff
# → should diff with previous if exists, or fall back to explore
```

- [ ] **Step 3: Commit any fixes from smoke testing**
