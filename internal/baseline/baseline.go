package baseline

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	baselineFile = ".baseline"
	previousFile = ".baseline-prev"
)

// Set records filename as the current baseline in resultsDir.
// The previous baseline is preserved for swap.
func Set(resultsDir, filename string) error {
	cur, _ := Get(resultsDir)
	if cur != "" && cur != filename {
		_ = os.WriteFile(filepath.Join(resultsDir, previousFile), []byte(cur), 0644)
	}
	return os.WriteFile(filepath.Join(resultsDir, baselineFile), []byte(filename), 0644)
}

// Get returns the current baseline filename, or empty if none is set.
func Get(resultsDir string) (string, error) {
	data, err := os.ReadFile(filepath.Join(resultsDir, baselineFile))
	if err != nil {
		return "", nil
	}
	return strings.TrimSpace(string(data)), nil
}

// GetPath returns the full path to the baseline result file.
func GetPath(resultsDir string) (string, error) {
	name, err := Get(resultsDir)
	if err != nil || name == "" {
		return "", err
	}
	path := filepath.Join(resultsDir, name)
	if _, err := os.Stat(path); err != nil {
		return "", fmt.Errorf("baseline file missing: %s", path)
	}
	return path, nil
}

// Swap switches the current and previous baselines.
func Swap(resultsDir string) error {
	cur, _ := Get(resultsDir)
	prev, _ := getPrevious(resultsDir)
	if prev == "" {
		return fmt.Errorf("no previous baseline to swap to")
	}
	_ = os.WriteFile(filepath.Join(resultsDir, baselineFile), []byte(prev), 0644)
	if cur != "" {
		_ = os.WriteFile(filepath.Join(resultsDir, previousFile), []byte(cur), 0644)
	}
	return nil
}

func getPrevious(resultsDir string) (string, error) {
	data, err := os.ReadFile(filepath.Join(resultsDir, previousFile))
	if err != nil {
		return "", nil
	}
	return strings.TrimSpace(string(data)), nil
}
