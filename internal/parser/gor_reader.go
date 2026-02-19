package parser

import (
	"bufio"
	"bytes"
	"fmt"
	"net/http"
	"os"
)

type GorReader struct {
	file   *os.File
	reader *bufio.Reader
}

// Ensure GorReader implements TrafficReader
var _ TrafficReader = (*GorReader)(nil)

func NewGorReader(filepath string) (*GorReader, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	return &GorReader{
		file:   file,
		reader: bufio.NewReaderSize(file, 1024*1024), // 1MB buffer
	}, nil
}

func (g *GorReader) Close() error {
	return g.file.Close()
}

func (g *GorReader) ReadRequests(chunkSize int) (<-chan *http.Request, <-chan error) {
	requests := make(chan *http.Request, chunkSize)
	errors := make(chan error, 1)

	go func() {
		defer close(requests)
		defer close(errors)

		scanner := bufio.NewScanner(g.reader)
		scanner.Buffer(make([]byte, 64*1024), 10*1024*1024)

		var httpLines []string
		inRequest := false
		
		for scanner.Scan() {
			line := scanner.Text()
			
			// Request ID line: "1 <id> <timestamp>"
			if len(line) > 0 && line[0] == '1' && len(line) > 40 {
				// Process previous request
				if len(httpLines) > 0 {
					if req := parseHTTPLines(httpLines); req != nil {
						requests <- req
					}
					httpLines = nil
				}
				inRequest = true
				continue
			}
			
			// Skip emoji separator
			if line == "🐵🙈🙉" {
				if len(httpLines) > 0 {
					if req := parseHTTPLines(httpLines); req != nil {
						requests <- req
					}
					httpLines = nil
				}
				inRequest = false
				continue
			}
			
			// Collect HTTP lines
			if inRequest && line != "" {
				httpLines = append(httpLines, line)
			}
		}
		
		// Process last request
		if len(httpLines) > 0 {
			if req := parseHTTPLines(httpLines); req != nil {
				requests <- req
			}
		}

		if err := scanner.Err(); err != nil {
			errors <- fmt.Errorf("scanner error: %w", err)
		}
	}()

	return requests, errors
}

func parseHTTPLines(lines []string) *http.Request {
	if len(lines) == 0 {
		return nil
	}
	
	// Build HTTP request with proper line endings
	var buf bytes.Buffer
	for _, line := range lines {
		buf.WriteString(line)
		buf.WriteString("\r\n")
	}
	buf.WriteString("\r\n") // Final empty line
	
	req, err := http.ReadRequest(bufio.NewReader(&buf))
	if err != nil {
		return nil
	}
	
	return req
}
