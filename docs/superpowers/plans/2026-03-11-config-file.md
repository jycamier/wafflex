# Config File Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `.wafflex.yaml` config file support so the `analyze` command can read defaults from a YAML file, with CLI flags as overrides.

**Architecture:** New `internal/config` package loads config from YAML using Viper. Root command gets a `--config` flag. Analyze command reads config values as defaults, CLI flags override them. Results auto-name with timestamps into `results-dir/`.

**Tech Stack:** Viper (config), Cobra (CLI)

---

## File Structure

| File | Role |
|------|------|
| `internal/config/config.go` | Config struct + Load function |
| `internal/config/config_test.go` | Tests for config loading |
| `cmd/root.go` | Add `--config` flag + Viper init |
| `cmd/analyze.go` | Use config as defaults, remove required flags |

---

### Task 1: Add Viper dependency

- [ ] **Step 1: Install Viper**

Run: `go get github.com/spf13/viper`

- [ ] **Step 2: Commit**

```bash
git add go.mod go.sum
git commit -m "chore: add viper dependency"
```

---

### Task 2: Create config package with tests

**Files:**
- Create: `internal/config/config.go`
- Create: `internal/config/config_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/config/config_test.go`:

```go
package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFromFile(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ".wafflex.yaml")
	content := []byte(`coraza-config: ./coraza.conf
traffic:
  type: gor
  file: ./traffic.gor
results-dir: ./output
`)
	if err := os.WriteFile(cfgPath, content, 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.CorazaConfig != "./coraza.conf" {
		t.Errorf("CorazaConfig = %q, want %q", cfg.CorazaConfig, "./coraza.conf")
	}
	if cfg.Traffic.Type != "gor" {
		t.Errorf("Traffic.Type = %q, want %q", cfg.Traffic.Type, "gor")
	}
	if cfg.Traffic.File != "./traffic.gor" {
		t.Errorf("Traffic.File = %q, want %q", cfg.Traffic.File, "./traffic.gor")
	}
	if cfg.ResultsDir != "./output" {
		t.Errorf("ResultsDir = %q, want %q", cfg.ResultsDir, "./output")
	}
}

func TestLoadFileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/.wafflex.yaml")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestLoadSearchOrder(t *testing.T) {
	// Create config in a temp dir simulating CWD
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ".wafflex.yaml")
	content := []byte(`coraza-config: ./found.conf
traffic:
  type: gor
  file: ./traffic.gor
results-dir: ./results
`)
	if err := os.WriteFile(cfgPath, content, 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Search(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.CorazaConfig != "./found.conf" {
		t.Errorf("CorazaConfig = %q, want %q", cfg.CorazaConfig, "./found.conf")
	}
}

func TestSearchNoConfigReturnsNil(t *testing.T) {
	dir := t.TempDir()
	cfg, err := Search(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg != nil {
		t.Error("expected nil config when no file found")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/config/ -v`
Expected: FAIL — package doesn't exist yet.

- [ ] **Step 3: Write the implementation**

Create `internal/config/config.go`:

```go
package config

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

const FileName = ".wafflex.yaml"

type TrafficConfig struct {
	Type string `mapstructure:"type"`
	File string `mapstructure:"file"`
}

type Config struct {
	CorazaConfig string        `mapstructure:"coraza-config"`
	Traffic      TrafficConfig `mapstructure:"traffic"`
	ResultsDir   string        `mapstructure:"results-dir"`
}

// Load reads config from an explicit file path.
func Load(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(path)
	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// Search looks for .wafflex.yaml in the given directory, then in $HOME.
// Returns nil (no error) if no config file is found.
func Search(dirs ...string) (*Config, error) {
	var searchPaths []string

	for _, d := range dirs {
		searchPaths = append(searchPaths, d)
	}

	home, err := os.UserHomeDir()
	if err == nil {
		searchPaths = append(searchPaths, home)
	}

	for _, dir := range searchPaths {
		path := filepath.Join(dir, FileName)
		if _, err := os.Stat(path); err == nil {
			return Load(path)
		}
	}

	return nil, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/config/ -v`
Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/config/
git commit -m "feat(config): add config package with Load and Search"
```

---

### Task 3: Wire config into root command

**Files:**
- Modify: `cmd/root.go`

- [ ] **Step 1: Add --config flag and Viper init to root.go**

Replace `cmd/root.go` with:

```go
package cmd

import (
	"fmt"
	"os"

	"github.com/jycamier/wafflex/internal/config"
	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use:   "wafflex",
		Short: "Test WAF rules with GoReplay traffic",
		Long:  `Wafflex replays HTTP traffic from GoReplay files through Coraza WAF and analyzes the results.`,
	}

	appConfig *config.Config
)

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().String("config", "", "config file (default: .wafflex.yaml in current dir or home)")
	rootCmd.AddCommand(analyzeCmd)
	rootCmd.AddCommand(exploreCmd)
	rootCmd.AddCommand(diffCmd)
}

func initConfig() {
	cfgPath, _ := rootCmd.PersistentFlags().GetString("config")

	if cfgPath != "" {
		cfg, err := config.Load(cfgPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config %s: %v\n", cfgPath, err)
			os.Exit(1)
		}
		appConfig = cfg
		return
	}

	cwd, err := os.Getwd()
	if err != nil {
		return
	}

	cfg, err := config.Search(cwd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: error reading config: %v\n", err)
		return
	}
	appConfig = cfg
}
```

- [ ] **Step 2: Verify it compiles**

Run: `go build ./...`
Expected: success.

- [ ] **Step 3: Commit**

```bash
git add cmd/root.go
git commit -m "feat(config): wire config loading into root command"
```

---

### Task 4: Update analyze command to use config

**Files:**
- Modify: `cmd/analyze.go`

- [ ] **Step 1: Update analyze.go to use config as defaults**

Replace the `init()` and beginning of `runAnalyze` in `cmd/analyze.go`:

The `init()` function removes `MarkFlagRequired` calls — flags become optional when config exists.

The `runAnalyze` function resolves values from: CLI flag (if set) > config file > error.

For `--output`, when not explicitly set and config has `results-dir`, generate a timestamped filename in that directory (creating it if needed).

The `traffic.type` from config is passed to `parser.NewTrafficReader` as an explicit reader type, avoiding extension-based guessing.

```go
func init() {
	analyzeCmd.Flags().StringP("gor-file", "g", "", "Path to traffic file (.gor or .custom)")
	analyzeCmd.Flags().StringP("coraza-config", "c", "", "Path to Coraza configuration file")
	analyzeCmd.Flags().StringP("output", "o", "", "Output JSON file path (default: auto-generated in results-dir)")
}

func runAnalyze(cmd *cobra.Command, args []string) error {
	gorFile, _ := cmd.Flags().GetString("gor-file")
	corazaConfig, _ := cmd.Flags().GetString("coraza-config")
	outputFile, _ := cmd.Flags().GetString("output")

	// Resolve from config if flags not set
	if gorFile == "" && appConfig != nil {
		gorFile = appConfig.Traffic.File
	}
	if corazaConfig == "" && appConfig != nil {
		corazaConfig = appConfig.CorazaConfig
	}

	// Validate required values
	if gorFile == "" {
		return fmt.Errorf("traffic file is required (use --gor-file flag or config file)")
	}
	if corazaConfig == "" {
		return fmt.Errorf("coraza config is required (use --coraza-config flag or config file)")
	}

	// Resolve output path
	if outputFile == "" {
		if appConfig != nil && appConfig.ResultsDir != "" {
			if err := os.MkdirAll(appConfig.ResultsDir, 0755); err != nil {
				return fmt.Errorf("failed to create results directory: %w", err)
			}
			timestamp := time.Now().Format("2006-01-02T15-04-05")
			outputFile = filepath.Join(appConfig.ResultsDir, timestamp+".json")
		} else {
			outputFile = "results.json"
		}
	}

	// ... rest of runAnalyze stays the same from the Validate inputs line
```

Also add `"path/filepath"` to the imports.

For the traffic reader creation, pass the explicit type from config:

```go
	// Open traffic file — use explicit type from config if available
	var reader parser.TrafficReader
	if appConfig != nil && appConfig.Traffic.Type != "" {
		reader, err = parser.NewTrafficReader(gorFile, parser.ReaderType(appConfig.Traffic.Type))
	} else {
		reader, err = parser.NewTrafficReader(gorFile)
	}
	if err != nil {
		return fmt.Errorf("failed to open traffic file: %w", err)
	}
	defer reader.Close()
```

- [ ] **Step 2: Verify it compiles**

Run: `go build ./...`
Expected: success.

- [ ] **Step 3: Run existing tests**

Run: `go test ./...`
Expected: all pass.

- [ ] **Step 4: Commit**

```bash
git add cmd/analyze.go
git commit -m "feat(config): use config file as defaults in analyze command"
```

---

### Task 5: Add example config file

**Files:**
- Create: `.wafflex.example.yaml`

- [ ] **Step 1: Create example config**

```yaml
# WAF engine configuration
coraza-config: ./coraza-test.conf

# Traffic source
traffic:
  type: gor
  file: ./traffic.gor

# Directory for analysis results (auto-generated filenames)
results-dir: ./results
```

- [ ] **Step 2: Commit**

```bash
git add .wafflex.example.yaml
git commit -m "docs(config): add example config file"
```
