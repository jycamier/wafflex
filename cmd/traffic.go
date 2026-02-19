package cmd

import (
	"fmt"

	"github.com/jycamier/wafflex/internal/fetcher"
	"github.com/jycamier/wafflex/internal/parser"
)

// openTrafficReader creates a TrafficReader from config and/or CLI flags.
// Returns the reader and an optional cleanup function (for fetched remote files).
func openTrafficReader(gorFile string) (parser.TrafficReader, func(), error) {
	trafficQuery := ""
	if appConfig != nil {
		trafficQuery = appConfig.Traffic.Query
	}

	isParquet := (appConfig != nil && appConfig.Traffic.Type == "parquet") || trafficQuery != ""

	if isParquet {
		if trafficQuery == "" {
			return nil, nil, fmt.Errorf("parquet query is required in config")
		}
		reader, err := parser.NewParquetReader(trafficQuery, appConfig.Traffic.Columns, appConfig.Traffic.DuckDB.Init...)
		if err != nil {
			return nil, nil, err
		}
		return reader, func() {}, nil
	}

	if gorFile == "" {
		return nil, nil, fmt.Errorf("traffic file is required (use --gor-file flag or config file)")
	}

	localTrafficFile, cleanup, err := fetcher.Resolve(gorFile)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch traffic file: %w", err)
	}

	var reader parser.TrafficReader
	if appConfig != nil && appConfig.Traffic.Type != "" {
		reader, err = parser.NewTrafficReader(localTrafficFile, parser.ReaderType(appConfig.Traffic.Type))
	} else {
		reader, err = parser.NewTrafficReader(localTrafficFile)
	}
	if err != nil {
		cleanup()
		return nil, nil, err
	}

	return reader, cleanup, nil
}
