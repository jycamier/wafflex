package hash

import (
	"crypto/sha256"
	"fmt"
	"os"

	"github.com/jycamier/wafflex/internal/config"
)

// TrafficSourceHash returns a 12-char hex hash identifying the traffic source.
// Parquet: hash of the SQL query. GOR/Custom: hash of the file content.
func TrafficSourceHash(cfg *config.Config) (string, error) {
	switch cfg.Traffic.Type {
	case "parquet":
		if cfg.Traffic.Query == "" {
			return "", fmt.Errorf("parquet query is required for traffic hash")
		}
		return shortHash([]byte(cfg.Traffic.Query)), nil
	default:
		if cfg.Traffic.File == "" {
			return "", fmt.Errorf("traffic file is required for traffic hash")
		}
		data, err := os.ReadFile(cfg.Traffic.File)
		if err != nil {
			return "", fmt.Errorf("failed to read traffic file: %w", err)
		}
		return shortHash(data), nil
	}
}

// RulesHash returns a 12-char hex hash of the Coraza config file content.
func RulesHash(corazaConfigPath string) (string, error) {
	data, err := os.ReadFile(corazaConfigPath)
	if err != nil {
		return "", fmt.Errorf("failed to read coraza config: %w", err)
	}
	return shortHash(data), nil
}

// ResultFileName returns the hash-based result file name.
func ResultFileName(trafficHash, rulesHash string) string {
	return trafficHash + "-" + rulesHash + ".json"
}

func shortHash(data []byte) string {
	h := sha256.Sum256(data)
	return fmt.Sprintf("%x", h)[:12]
}
