package cache

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	envCacheDir    = "WAFFLEX_CACHE_DIR"
	defaultBaseDir = ".wafflex"
	cacheSubDir    = "cache"
)

// Dir returns the cache directory path.
// Priority: WAFFLEX_CACHE_DIR env var > ~/.wafflex/cache/
func Dir() (string, error) {
	if dir := os.Getenv(envCacheDir); dir != "" {
		return dir, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	return filepath.Join(home, defaultBaseDir, cacheSubDir), nil
}

// queryHash returns the hex-encoded SHA256 hash for a query.
func queryHash(query string) string {
	hash := sha256.Sum256([]byte(query))
	return fmt.Sprintf("%x", hash)
}

// QueryKey returns a deterministic cache filename for a given SQL query.
func QueryKey(query string) string {
	return queryHash(query) + ".parquet"
}

// queryMetaKey returns the metadata filename for a given query.
func queryMetaKey(query string) string {
	return queryHash(query) + ".query"
}

// Lookup checks if a cached parquet file exists for the given query.
// If maxAge is provided and > 0, entries older than maxAge are considered stale
// and removed automatically.
// Returns the path if found and fresh, empty string otherwise.
func Lookup(query string, maxAge ...time.Duration) (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}

	path := filepath.Join(dir, QueryKey(query))
	info, err := os.Stat(path)
	if err != nil {
		return "", nil
	}

	if len(maxAge) > 0 && maxAge[0] > 0 && time.Since(info.ModTime()) > maxAge[0] {
		removeEntry(dir, query)
		return "", nil
	}

	return path, nil
}

// removeEntry deletes both the cached parquet and its query metadata file.
func removeEntry(dir, query string) {
	os.Remove(filepath.Join(dir, QueryKey(query)))
	os.Remove(filepath.Join(dir, queryMetaKey(query)))
}

// Path returns the full path where a cached result should be stored.
// It also saves the original query text alongside for listing purposes.
func Path(query string) (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Save query text for listing
	metaPath := filepath.Join(dir, queryMetaKey(query))
	_ = os.WriteFile(metaPath, []byte(query), 0644)

	return filepath.Join(dir, QueryKey(query)), nil
}

// Entry represents a cached query result.
type Entry struct {
	Hash      string
	Query     string
	ParquetPath string
	Size      int64
}

// List returns all cached entries with their original queries.
func List() ([]Entry, error) {
	dir, err := Dir()
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, nil
	}

	matches, err := filepath.Glob(filepath.Join(dir, "*.query"))
	if err != nil {
		return nil, err
	}

	var entries []Entry
	for _, metaPath := range matches {
		queryBytes, err := os.ReadFile(metaPath)
		if err != nil {
			continue
		}

		base := filepath.Base(metaPath)
		hash := base[:len(base)-len(".query")]
		parquetPath := filepath.Join(dir, hash+".parquet")

		var size int64
		if info, err := os.Stat(parquetPath); err == nil {
			size = info.Size()
		}

		entries = append(entries, Entry{
			Hash:        hash[:12],
			Query:       string(queryBytes),
			ParquetPath: parquetPath,
			Size:        size,
		})
	}

	return entries, nil
}

// Clear removes all cached parquet files.
func Clear() error {
	dir, err := Dir()
	if err != nil {
		return err
	}
	return ClearDir(dir)
}

// ClearDir removes all files in the given directory.
func ClearDir(dir string) error {
	return os.RemoveAll(dir)
}
