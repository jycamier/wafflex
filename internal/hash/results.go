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
