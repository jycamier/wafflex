package parser

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

// CustomReader reads traffic from a custom JSON format
// Format: one JSON object per line with fields: method, url, headers, body
type CustomReader struct {
	file   *os.File
	reader *bufio.Reader
}

// Ensure CustomReader implements TrafficReader
var _ TrafficReader = (*CustomReader)(nil)

// CustomRequest represents a request in the custom format
type CustomRequest struct {
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
}

func NewCustomReader(filepath string) (*CustomReader, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	return &CustomReader{
		file:   file,
		reader: bufio.NewReaderSize(file, 1024*1024), // 1MB buffer
	}, nil
}

func (c *CustomReader) Close() error {
	return c.file.Close()
}

func (c *CustomReader) ReadRequests(chunkSize int) (<-chan *http.Request, <-chan error) {
	requests := make(chan *http.Request, chunkSize)
	errors := make(chan error, 1)

	go func() {
		defer close(requests)
		defer close(errors)

		scanner := bufio.NewScanner(c.reader)
		scanner.Buffer(make([]byte, 64*1024), 10*1024*1024) // 10MB max token size

		for scanner.Scan() {
			line := scanner.Bytes()
			
			var customReq CustomRequest
			if err := json.Unmarshal(line, &customReq); err != nil {
				continue // Skip invalid lines
			}

			// Convert to http.Request
			req, err := customReq.ToHTTPRequest()
			if err != nil {
				continue // Skip invalid requests
			}

			requests <- req
		}

		if err := scanner.Err(); err != nil {
			errors <- fmt.Errorf("scanner error: %w", err)
		}
	}()

	return requests, errors
}

func (c *CustomRequest) ToHTTPRequest() (*http.Request, error) {
	// Create request
	var bodyReader *bytes.Reader
	if c.Body != "" {
		bodyReader = bytes.NewReader([]byte(c.Body))
	} else {
		bodyReader = bytes.NewReader([]byte{})
	}

	req, err := http.NewRequest(c.Method, c.URL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	for k, v := range c.Headers {
		req.Header.Set(k, v)
	}

	return req, nil
}
