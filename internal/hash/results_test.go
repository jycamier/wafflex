package hash

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFindResultsMatchesPrefix(t *testing.T) {
	dir := t.TempDir()

	os.WriteFile(filepath.Join(dir, "aabbccddee11-111111111111.json"), []byte("{}"), 0644)
	os.WriteFile(filepath.Join(dir, "aabbccddee11-222222222222.json"), []byte("{}"), 0644)
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

	f1 := filepath.Join(dir, "aabbccddee11-111111111111.json")
	f2 := filepath.Join(dir, "aabbccddee11-222222222222.json")
	os.WriteFile(f1, []byte("{}"), 0644)
	time.Sleep(10 * time.Millisecond)
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

func TestParseResultFileName(t *testing.T) {
	rf, ok := ParseResultFileName("cb4004b7dd82-a1b2c3d4e5f6.json")
	if !ok {
		t.Fatal("expected parse to succeed")
	}
	if rf.TrafficHash != "cb4004b7dd82" {
		t.Errorf("TrafficHash = %q, want %q", rf.TrafficHash, "cb4004b7dd82")
	}
	if rf.RulesHash != "a1b2c3d4e5f6" {
		t.Errorf("RulesHash = %q, want %q", rf.RulesHash, "a1b2c3d4e5f6")
	}
}

func TestParseResultFileNameInvalid(t *testing.T) {
	_, ok := ParseResultFileName("2026-03-11T17-11-18.json")
	if ok {
		t.Error("expected parse to fail for timestamp-named file")
	}
}
