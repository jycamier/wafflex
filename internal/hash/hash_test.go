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
