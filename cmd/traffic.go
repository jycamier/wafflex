package cmd

import (
	"log/slog"
	"os"
	"time"

	"github.com/jycamier/wafflex/internal/config"
	"github.com/jycamier/wafflex/internal/fetcher"
	"github.com/jycamier/wafflex/internal/parser"
)

const defaultCacheTTL = 24 * time.Hour

// openTrafficReader creates a TrafficReader from config and/or CLI flags.
// Returns the reader and an optional cleanup function (for fetched remote files).
func openTrafficReader(gorFile string) (parser.TrafficReader, func()) {
	trafficQuery := ""
	if appConfig != nil {
		trafficQuery = appConfig.Traffic.Query
	}

	isParquet := (appConfig != nil && appConfig.Traffic.Type == "parquet") || trafficQuery != ""

	if isParquet {
		if trafficQuery == "" {
			slog.Error("parquet query is required in config")
			os.Exit(1)
		}
		reader, err := parser.NewParquetReader(trafficQuery, appConfig.Traffic.Columns, appConfig.Traffic.DuckDB.Init...)
		if err != nil {
			slog.Error("failed to create parquet reader", "error", err)
			os.Exit(1)
		}
		reader.SetCacheTTL(parseCacheTTL(appConfig))
		return reader, func() {}
	}

	if gorFile == "" {
		slog.Error("traffic file is required (use --gor-file flag or config file)")
		os.Exit(1)
	}

	localTrafficFile, cleanup, err := fetcher.Resolve(gorFile)
	if err != nil {
		slog.Error("failed to fetch traffic file", "error", err)
		os.Exit(1)
	}

	var reader parser.TrafficReader
	if appConfig != nil && appConfig.Traffic.Type != "" {
		reader, err = parser.NewTrafficReader(localTrafficFile, parser.ReaderType(appConfig.Traffic.Type))
	} else {
		reader, err = parser.NewTrafficReader(localTrafficFile)
	}
	if err != nil {
		cleanup()
		slog.Error("failed to create traffic reader", "error", err)
		os.Exit(1)
	}

	return reader, cleanup
}

func parseCacheTTL(cfg *config.Config) time.Duration {
	if cfg == nil || cfg.CacheTTL == "" {
		return defaultCacheTTL
	}
	d, err := time.ParseDuration(cfg.CacheTTL)
	if err != nil {
		slog.Warn("invalid cache-ttl, using default", "value", cfg.CacheTTL, "default", defaultCacheTTL)
		return defaultCacheTTL
	}
	return d
}
