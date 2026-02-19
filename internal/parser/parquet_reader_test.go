package parser

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jycamier/wafflex/internal/config"
)

func TestNewParquetReaderValidation(t *testing.T) {
	cols := config.ColumnMapping{Method: "m", Path: "p"}

	_, err := NewParquetReader("", cols)
	if err == nil {
		t.Error("expected error for empty query")
	}

	_, err = NewParquetReader("SELECT 1", config.ColumnMapping{})
	if err == nil {
		t.Error("expected error for missing method/path mapping")
	}

	_, err = NewParquetReader("SELECT 1", config.ColumnMapping{Method: "m"})
	if err == nil {
		t.Error("expected error for missing path mapping")
	}
}

func TestParquetReaderWithLocalFile(t *testing.T) {
	t.Setenv("WAFFLEX_CACHE_DIR", t.TempDir())
	// Create a small parquet file via DuckDB
	dir := t.TempDir()
	parquetPath := filepath.Join(dir, "test.parquet")

	reader, err := NewParquetReader(
		"SELECT 1", // dummy query to init DuckDB
		config.ColumnMapping{Method: "method", Path: "path"},
	)
	if err != nil {
		t.Fatalf("failed to create reader: %v", err)
	}

	// Use the DuckDB instance to create a test parquet file
	_, err = reader.db.Exec(`
		COPY (
			SELECT
				'GET' AS method,
				'/api/users' AS path,
				'example.com' AS host,
				'HTTP/1.1' AS proto,
				'{"Content-Type":"application/json"}' AS headers,
				'1.2.3.4' AS client_ip
			UNION ALL
			SELECT
				'POST' AS method,
				'/api/login' AS path,
				'example.com' AS host,
				'HTTP/1.1' AS proto,
				'{"Content-Type":"application/x-www-form-urlencoded"}' AS headers,
				'5.6.7.8' AS client_ip
		) TO '` + parquetPath + `' (FORMAT PARQUET)
	`)
	if err != nil {
		t.Fatalf("failed to create test parquet: %v", err)
	}
	reader.Close()

	// Verify the file exists
	if _, err := os.Stat(parquetPath); err != nil {
		t.Fatalf("parquet file not created: %v", err)
	}

	// Now read it back with column mapping
	reader2, err := NewParquetReader(
		"SELECT * FROM read_parquet('"+parquetPath+"')",
		config.ColumnMapping{
			Method:   "method",
			Path:     "path",
			Host:     "host",
			Proto:    "proto",
			Headers:  "headers",
			ClientIP: "client_ip",
		},
	)
	if err != nil {
		t.Fatalf("failed to create reader: %v", err)
	}
	defer reader2.Close()

	requests, errChan := reader2.ReadRequests(100)

	var count int
	for req := range requests {
		count++
		if req.Method != "GET" && req.Method != "POST" {
			t.Errorf("unexpected method: %s", req.Method)
		}
		if req.Host != "example.com" {
			t.Errorf("unexpected host: %s", req.Host)
		}
		if req.Proto != "HTTP/1.1" {
			t.Errorf("unexpected proto: %s", req.Proto)
		}
		if req.Header.Get("Content-Type") == "" {
			t.Error("expected Content-Type header to be set")
		}
		if req.RemoteAddr == "" {
			t.Error("expected RemoteAddr to be set")
		}
	}

	if err := <-errChan; err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if count != 2 {
		t.Errorf("expected 2 requests, got %d", count)
	}
}

func TestParquetReaderMinimalMapping(t *testing.T) {
	t.Setenv("WAFFLEX_CACHE_DIR", t.TempDir())
	dir := t.TempDir()
	parquetPath := filepath.Join(dir, "minimal.parquet")

	// Create parquet with only method and path
	setup, err := NewParquetReader("SELECT 1", config.ColumnMapping{Method: "m", Path: "p"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = setup.db.Exec(`
		COPY (SELECT 'GET' AS m, '/health' AS p)
		TO '` + parquetPath + `' (FORMAT PARQUET)
	`)
	if err != nil {
		t.Fatal(err)
	}
	setup.Close()

	reader, err := NewParquetReader(
		"SELECT * FROM read_parquet('"+parquetPath+"')",
		config.ColumnMapping{Method: "m", Path: "p"},
	)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()

	requests, errChan := reader.ReadRequests(100)

	req := <-requests
	if req == nil {
		t.Fatal("expected a request")
	}
	if req.Method != "GET" {
		t.Errorf("method = %q, want GET", req.Method)
	}
	if req.URL.Path != "/health" {
		t.Errorf("path = %q, want /health", req.URL.Path)
	}

	// Drain
	for range requests {
	}
	if err := <-errChan; err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
