package fetcher

import (
	"fmt"
	"os"
)

// LocalFetcher handles local file paths (passthrough).
type LocalFetcher struct{}

func (f *LocalFetcher) Fetch(uri string) (string, func(), error) {
	if _, err := os.Stat(uri); os.IsNotExist(err) {
		return "", nil, fmt.Errorf("local file does not exist: %s", uri)
	}
	return uri, noopCleanup, nil
}
