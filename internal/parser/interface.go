package parser

import "net/http"

// TrafficReader is an interface for reading HTTP traffic from various sources
type TrafficReader interface {
	// ReadRequests returns a channel of HTTP requests and a channel for errors
	// The chunkSize parameter is a hint for batching (may be ignored by implementations)
	ReadRequests(chunkSize int) (<-chan *http.Request, <-chan error)
	
	// Close closes the underlying reader and releases resources
	Close() error
}
