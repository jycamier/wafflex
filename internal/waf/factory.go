package waf

import (
	"fmt"
	"path/filepath"
	"strings"
)

// EngineType represents the type of WAF engine
type EngineType string

const (
	EngineTypeCoraza     EngineType = "coraza"
	EngineTypeModSecurity EngineType = "modsecurity"
	EngineTypeCustom     EngineType = "custom"
	// Add more types here as needed
)

// NewWAFEngine creates a WAFEngine based on the config file extension or explicit type
func NewWAFEngine(configPath string, engineType ...EngineType) (WAFEngine, error) {
	var eType EngineType
	
	// If type is explicitly provided, use it
	if len(engineType) > 0 {
		eType = engineType[0]
	} else {
		// Auto-detect from file extension or content
		ext := strings.ToLower(filepath.Ext(configPath))
		switch ext {
		case ".conf":
			// Default to Coraza for .conf files
			eType = EngineTypeCoraza
		case ".json":
			// JSON config could be custom
			eType = EngineTypeCustom
		case ".yaml", ".yml":
			// YAML config could be custom
			eType = EngineTypeCustom
		default:
			// Default to Coraza
			eType = EngineTypeCoraza
		}
	}
	
	// Create the appropriate engine
	switch eType {
	case EngineTypeCoraza:
		return NewEngine(configPath)
	case EngineTypeModSecurity:
		return NewModSecurityEngine(configPath)
	case EngineTypeCustom:
		return NewCustomEngine(configPath)
	default:
		return nil, fmt.Errorf("unsupported engine type: %s", eType)
	}
}
