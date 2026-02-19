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

func TestSearchFindsConfigInDir(t *testing.T) {
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
