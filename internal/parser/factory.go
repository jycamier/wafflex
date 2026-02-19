package parser

import (
	"fmt"
	"path/filepath"
	"strings"
)

// ReaderType represents the type of traffic file
type ReaderType string

const (
	ReaderTypeGor     ReaderType = "gor"
	ReaderTypeCustom  ReaderType = "custom"
	ReaderTypeParquet ReaderType = "parquet"
)

// NewTrafficReader creates a TrafficReader based on the file extension or explicit type
func NewTrafficReader(filePath string, readerType ...ReaderType) (TrafficReader, error) {
	var rType ReaderType
	
	// If type is explicitly provided, use it
	if len(readerType) > 0 {
		rType = readerType[0]
	} else {
		// Auto-detect from file extension
		ext := strings.ToLower(filepath.Ext(filePath))
		switch ext {
		case ".gor":
			rType = ReaderTypeGor
		case ".custom":
			rType = ReaderTypeCustom
		default:
			return nil, fmt.Errorf("unsupported file extension: %s (use .gor or .custom)", ext)
		}
	}
	
	// Create the appropriate reader
	switch rType {
	case ReaderTypeGor:
		return NewGorReader(filePath)
	case ReaderTypeCustom:
		return NewCustomReader(filePath)
	default:
		return nil, fmt.Errorf("unsupported reader type: %s", rType)
	}
}
