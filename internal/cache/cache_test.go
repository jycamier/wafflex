package cache

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestQueryKeyDeterministic(t *testing.T) {
	query := "SELECT * FROM read_parquet('s3://bucket/data.parquet')"
	key1 := QueryKey(query)
	key2 := QueryKey(query)

	if key1 != key2 {
		t.Errorf("same query produced different keys: %s vs %s", key1, key2)
	}
}

func TestQueryKeyDifferentQueries(t *testing.T) {
	key1 := QueryKey("SELECT * FROM a")
	key2 := QueryKey("SELECT * FROM b")

	if key1 == key2 {
		t.Error("different queries produced same key")
	}
}

func TestDirFromEnv(t *testing.T) {
	customDir := t.TempDir()
	t.Setenv(envCacheDir, customDir)

	dir, err := Dir()
	if err != nil {
		t.Fatal(err)
	}
	if dir != customDir {
		t.Errorf("dir = %q, want %q", dir, customDir)
	}
}

func TestDirDefault(t *testing.T) {
	t.Setenv(envCacheDir, "")

	dir, err := Dir()
	if err != nil {
		t.Fatal(err)
	}

	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, defaultBaseDir, cacheSubDir)
	if dir != expected {
		t.Errorf("dir = %q, want %q", dir, expected)
	}
}

func TestLookupMiss(t *testing.T) {
	t.Setenv(envCacheDir, t.TempDir())

	path, err := Lookup("some query")
	if err != nil {
		t.Fatal(err)
	}
	if path != "" {
		t.Errorf("expected empty path for cache miss, got %q", path)
	}
}

func TestLookupHit(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(envCacheDir, dir)

	query := "SELECT * FROM test"
	cachePath := filepath.Join(dir, QueryKey(query))
	if err := os.WriteFile(cachePath, []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}

	path, err := Lookup(query)
	if err != nil {
		t.Fatal(err)
	}
	if path != cachePath {
		t.Errorf("path = %q, want %q", path, cachePath)
	}
}

func TestPathCreatesDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "cache")
	t.Setenv(envCacheDir, dir)

	path, err := Path("some query")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Error("expected cache directory to be created")
	}

	expected := filepath.Join(dir, QueryKey("some query"))
	if path != expected {
		t.Errorf("path = %q, want %q", path, expected)
	}
}

func TestClear(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(envCacheDir, dir)

	// Create a fake cached file
	if err := os.WriteFile(filepath.Join(dir, "test.parquet"), []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := Clear(); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Error("expected cache directory to be removed")
	}
}

func TestClearNonExistent(t *testing.T) {
	t.Setenv(envCacheDir, filepath.Join(t.TempDir(), "nonexistent"))

	if err := Clear(); err != nil {
		t.Fatalf("clear on non-existent dir should not error: %v", err)
	}
}

func TestLookupNoTTL(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(envCacheDir, dir)

	query := "SELECT * FROM test"
	cachePath := filepath.Join(dir, QueryKey(query))
	if err := os.WriteFile(cachePath, []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}

	// No TTL: should always hit
	path, err := Lookup(query)
	if err != nil {
		t.Fatal(err)
	}
	if path != cachePath {
		t.Errorf("expected cache hit without TTL, got empty")
	}
}

func TestLookupZeroTTL(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(envCacheDir, dir)

	query := "SELECT * FROM test"
	cachePath := filepath.Join(dir, QueryKey(query))
	if err := os.WriteFile(cachePath, []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}

	// TTL of 0: should always hit (no expiry)
	path, err := Lookup(query, 0)
	if err != nil {
		t.Fatal(err)
	}
	if path != cachePath {
		t.Errorf("expected cache hit with zero TTL, got empty")
	}
}

func TestLookupFreshEntry(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(envCacheDir, dir)

	query := "SELECT * FROM test"
	cachePath := filepath.Join(dir, QueryKey(query))
	if err := os.WriteFile(cachePath, []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}

	// TTL of 1 hour: file just created, should hit
	path, err := Lookup(query, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if path != cachePath {
		t.Errorf("expected cache hit for fresh entry, got empty")
	}
}

func TestLookupStaleEntry(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(envCacheDir, dir)

	query := "SELECT * FROM test"
	cachePath := filepath.Join(dir, QueryKey(query))
	metaPath := filepath.Join(dir, queryMetaKey(query))
	if err := os.WriteFile(cachePath, []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(metaPath, []byte(query), 0644); err != nil {
		t.Fatal(err)
	}

	// Backdate the file to 2 hours ago
	past := time.Now().Add(-2 * time.Hour)
	os.Chtimes(cachePath, past, past)

	// TTL of 1 hour: file is stale, should miss and remove
	path, err := Lookup(query, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if path != "" {
		t.Errorf("expected cache miss for stale entry, got %q", path)
	}

	// Both files should be removed
	if _, err := os.Stat(cachePath); !os.IsNotExist(err) {
		t.Error("expected stale parquet file to be removed")
	}
	if _, err := os.Stat(metaPath); !os.IsNotExist(err) {
		t.Error("expected stale metadata file to be removed")
	}
}
