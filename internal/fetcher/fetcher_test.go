package fetcher

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLocalFetcherExistingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.gor")
	if err := os.WriteFile(path, []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}

	localPath, cleanup, err := Resolve(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer cleanup()

	if localPath != path {
		t.Errorf("localPath = %q, want %q", localPath, path)
	}
}

func TestLocalFetcherMissingFile(t *testing.T) {
	_, _, err := Resolve("/nonexistent/file.gor")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestParseS3URI(t *testing.T) {
	tests := []struct {
		uri        string
		wantBucket string
		wantKey    string
		wantErr    bool
	}{
		{"ref+s3://my-bucket/path/to/file.gor", "my-bucket", "path/to/file.gor", false},
		{"ref+s3://bucket/key", "bucket", "key", false},
		{"ref+s3://bucket/", "", "", true},
		{"ref+s3://", "", "", true},
		{"ref+s3://bucket-only", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.uri, func(t *testing.T) {
			bucket, key, err := parseS3URI(tt.uri)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if bucket != tt.wantBucket {
				t.Errorf("bucket = %q, want %q", bucket, tt.wantBucket)
			}
			if key != tt.wantKey {
				t.Errorf("key = %q, want %q", key, tt.wantKey)
			}
		})
	}
}

func TestResolveDetectsS3Prefix(t *testing.T) {
	// We can't actually test S3 download without AWS, but we can verify
	// that the factory routes to S3Fetcher (which will fail with no AWS config)
	_, _, err := Resolve("ref+s3://bucket/key")
	if err == nil {
		t.Skip("AWS credentials available, skipping negative test")
	}
	// Error should mention AWS or S3, not "local file does not exist"
	if err.Error() == "local file does not exist: ref+s3://bucket/key" {
		t.Error("S3 URI was routed to LocalFetcher instead of S3Fetcher")
	}
}
