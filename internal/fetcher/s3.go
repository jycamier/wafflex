package fetcher

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3Fetcher downloads files from S3 using ref+s3://bucket/key URIs.
// Authentication uses the default AWS SDK credential chain.
type S3Fetcher struct{}

func (f *S3Fetcher) Fetch(uri string) (string, func(), error) {
	bucket, key, err := parseS3URI(uri)
	if err != nil {
		return "", nil, err
	}

	ctx := context.Background()

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return "", nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	client := s3.NewFromConfig(cfg)

	output, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &bucket,
		Key:    &key,
	})
	if err != nil {
		return "", nil, fmt.Errorf("failed to download s3://%s/%s: %w", bucket, key, err)
	}
	defer output.Body.Close()

	// Create temp file preserving the original extension
	ext := filepath.Ext(key)
	tmpFile, err := os.CreateTemp("", "wafflex-*"+ext)
	if err != nil {
		return "", nil, fmt.Errorf("failed to create temp file: %w", err)
	}

	if _, err := io.Copy(tmpFile, output.Body); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return "", nil, fmt.Errorf("failed to write temp file: %w", err)
	}
	tmpFile.Close()

	cleanup := func() {
		os.Remove(tmpFile.Name())
	}

	return tmpFile.Name(), cleanup, nil
}
