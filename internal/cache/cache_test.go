package cache

import (
	"os"
	"path/filepath"
	"testing"
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
