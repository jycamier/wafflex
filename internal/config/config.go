package config

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

const FileName = ".wafflex.yaml"

type ColumnMapping struct {
	Method    string `mapstructure:"method"`
	Path      string `mapstructure:"path"`
	Host      string `mapstructure:"host"`
	Proto     string `mapstructure:"proto"`
	Headers   string `mapstructure:"headers"`
	Body      string `mapstructure:"body"`
	ClientIP  string `mapstructure:"client_ip"`
	Timestamp string `mapstructure:"timestamp"`
}

type DuckDBConfig struct {
	Init []string `mapstructure:"init"`
}

type TrafficConfig struct {
	Type    string        `mapstructure:"type"`
	File    string        `mapstructure:"file"`
	Query   string        `mapstructure:"query"`
	Columns ColumnMapping `mapstructure:"columns"`
	DuckDB  DuckDBConfig  `mapstructure:"duckdb"`
}

type Config struct {
	CorazaConfig string        `mapstructure:"coraza-config"`
	Traffic      TrafficConfig `mapstructure:"traffic"`
	ResultsDir   string        `mapstructure:"results-dir"`
	CacheTTL     string        `mapstructure:"cache-ttl"`
}

// Load reads config from an explicit file path.
func Load(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(path)
	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// Search looks for .wafflex.yaml in the given directories, then in $HOME.
// Returns nil (no error) if no config file is found.
func Search(dirs ...string) (*Config, error) {
	var searchPaths []string

	for _, d := range dirs {
		searchPaths = append(searchPaths, d)
	}

	home, err := os.UserHomeDir()
	if err == nil {
		searchPaths = append(searchPaths, home)
	}

	for _, dir := range searchPaths {
		path := filepath.Join(dir, FileName)
		if _, err := os.Stat(path); err == nil {
			return Load(path)
		}
	}

	return nil, nil
}
