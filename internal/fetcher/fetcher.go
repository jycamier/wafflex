package fetcher

import (
	"fmt"
	"strings"
)

// Fetcher downloads a file from a URI and returns a local path.
type Fetcher interface {
	Fetch(uri string) (localPath string, cleanup func(), err error)
}

const (
	s3Prefix = "ref+s3://"
)

// Resolve detects the URI scheme and fetches the file locally.
// For local paths, it returns the path as-is with a no-op cleanup.
// For ref+s3:// URIs, it downloads to a temp file.
func Resolve(uri string) (string, func(), error) {
	switch {
	case strings.HasPrefix(uri, s3Prefix):
		f := &S3Fetcher{}
		return f.Fetch(uri)
	default:
		f := &LocalFetcher{}
		return f.Fetch(uri)
	}
}

// noopCleanup does nothing.
func noopCleanup() {}

// parseS3URI extracts bucket and key from a ref+s3://bucket/key URI.
func parseS3URI(uri string) (bucket, key string, err error) {
	path := strings.TrimPrefix(uri, s3Prefix)
	if path == "" {
		return "", "", fmt.Errorf("empty S3 URI")
	}
	parts := strings.SplitN(path, "/", 2)
	if len(parts) < 2 || parts[1] == "" {
		return "", "", fmt.Errorf("invalid S3 URI %q: must be ref+s3://bucket/key", uri)
	}
	return parts[0], parts[1], nil
}
