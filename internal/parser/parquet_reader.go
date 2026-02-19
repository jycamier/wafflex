package parser

import (
	"bytes"
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"

	"github.com/duckdb/duckdb-go/v2"
	"github.com/jycamier/wafflex/internal/cache"
	"github.com/jycamier/wafflex/internal/config"
)

// ParquetReader reads HTTP requests from Parquet files via DuckDB queries.
type ParquetReader struct {
	connector *duckdb.Connector
	db        *sql.DB
	query     string
	columns   config.ColumnMapping
}

// NewParquetReader creates a reader that executes a DuckDB SQL query
// and maps result columns to HTTP requests using the provided column mapping.
// initStatements are executed at connection boot via DuckDB's native connector init.
// Env vars in ${VAR} syntax are expanded before execution.
func NewParquetReader(query string, columns config.ColumnMapping, initStatements ...string) (*ParquetReader, error) {
	if query == "" {
		return nil, fmt.Errorf("parquet query is required")
	}
	if columns.Method == "" || columns.Path == "" {
		return nil, fmt.Errorf("column mapping for method and path is required")
	}

	connector, err := duckdb.NewConnector("", func(execer driver.ExecerContext) error {
		for _, stmt := range initStatements {
			expanded := os.ExpandEnv(stmt)
			if _, err := execer.ExecContext(context.Background(), expanded, nil); err != nil {
				return fmt.Errorf("init statement %q: %w", expanded, err)
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create DuckDB connector: %w", err)
	}

	db := sql.OpenDB(connector)

	return &ParquetReader{
		connector: connector,
		db:        db,
		query:     query,
		columns:   columns,
	}, nil
}

func (r *ParquetReader) ReadRequests(chunkSize int) (<-chan *http.Request, <-chan error) {
	requests := make(chan *http.Request, chunkSize)
	errChan := make(chan error, 1)

	go func() {
		defer close(requests)
		defer close(errChan)

		// Resolve the query to execute: use cache if available, otherwise run original query and cache result
		queryToExecute, err := r.resolveQuery()
		if err != nil {
			errChan <- err
			return
		}

		rows, err := r.db.QueryContext(context.Background(), queryToExecute)
		if err != nil {
			errChan <- fmt.Errorf("failed to execute parquet query: %w", err)
			return
		}
		defer rows.Close()

		colNames, err := rows.Columns()
		if err != nil {
			errChan <- fmt.Errorf("failed to get columns: %w", err)
			return
		}

		colIndex := buildColumnIndex(colNames)

		for rows.Next() {
			values := make([]interface{}, len(colNames))
			valuePtrs := make([]interface{}, len(colNames))
			for i := range values {
				valuePtrs[i] = &values[i]
			}

			if err := rows.Scan(valuePtrs...); err != nil {
				errChan <- fmt.Errorf("failed to scan row: %w", err)
				return
			}

			req, err := r.rowToRequest(values, colIndex)
			if err != nil {
				continue
			}
			requests <- req
		}

		if err := rows.Err(); err != nil {
			errChan <- fmt.Errorf("row iteration error: %w", err)
		}
	}()

	return requests, errChan
}

func (r *ParquetReader) Close() error {
	err := r.db.Close()
	if r.connector != nil {
		r.connector.Close()
	}
	return err
}

// resolveQuery checks the cache for a stored result.
// On cache hit: returns a query that reads from the cached local parquet file.
// On cache miss: executes the original query, saves results to cache, returns a query reading the cache.
func (r *ParquetReader) resolveQuery() (string, error) {
	cached, err := cache.Lookup(r.query)
	if err != nil {
		return "", fmt.Errorf("cache lookup failed: %w", err)
	}
	if cached != "" {
		slog.Info("cache hit", "path", cached)
		return "SELECT * FROM read_parquet('" + cached + "')", nil
	}

	// Cache miss: execute query and store result
	cachePath, err := cache.Path(r.query)
	if err != nil {
		// Cache unavailable, fall back to direct query
		slog.Warn("cache unavailable, querying directly")
		return r.query, nil
	}

	slog.Info("cache miss, executing query and caching result")
	copyQuery := fmt.Sprintf("COPY (%s) TO '%s' (FORMAT PARQUET)", r.query, cachePath)
	if _, err := r.db.ExecContext(context.Background(), copyQuery); err != nil {
		// If caching fails, fall back to direct query
		slog.Warn("failed to cache result", "error", err)
		return r.query, nil
	}

	slog.Info("cached result", "path", cachePath)
	return "SELECT * FROM read_parquet('" + cachePath + "')", nil
}

// buildColumnIndex maps column names to their index position.
func buildColumnIndex(colNames []string) map[string]int {
	idx := make(map[string]int, len(colNames))
	for i, name := range colNames {
		idx[name] = i
	}
	return idx
}

// rowToRequest converts a row of values into an http.Request using the column mapping.
func (r *ParquetReader) rowToRequest(values []interface{}, colIndex map[string]int) (*http.Request, error) {
	method := getStringValue(values, colIndex, r.columns.Method)
	path := getStringValue(values, colIndex, r.columns.Path)

	if method == "" || path == "" {
		return nil, fmt.Errorf("missing method or path")
	}

	reqURL, err := url.Parse(path)
	if err != nil {
		return nil, fmt.Errorf("invalid path %q: %w", path, err)
	}

	var body io.Reader
	if r.columns.Body != "" {
		bodyData := getBytesValue(values, colIndex, r.columns.Body)
		if len(bodyData) > 0 {
			body = bytes.NewReader(bodyData)
		}
	}

	req, err := http.NewRequest(method, reqURL.String(), body)
	if err != nil {
		return nil, err
	}

	if r.columns.Host != "" {
		req.Host = getStringValue(values, colIndex, r.columns.Host)
	}
	if r.columns.Proto != "" {
		req.Proto = getStringValue(values, colIndex, r.columns.Proto)
	}
	if r.columns.ClientIP != "" {
		req.RemoteAddr = getStringValue(values, colIndex, r.columns.ClientIP)
	}
	if r.columns.Headers != "" {
		headersJSON := getStringValue(values, colIndex, r.columns.Headers)
		if headersJSON != "" {
			var headerMap map[string]string
			if err := json.Unmarshal([]byte(headersJSON), &headerMap); err == nil {
				for k, v := range headerMap {
					req.Header.Set(k, v)
				}
			}
		}
	}

	return req, nil
}

func getStringValue(values []interface{}, colIndex map[string]int, colName string) string {
	idx, ok := colIndex[colName]
	if !ok {
		return ""
	}
	v := values[idx]
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case []byte:
		return string(val)
	default:
		return fmt.Sprintf("%v", val)
	}
}

func getBytesValue(values []interface{}, colIndex map[string]int, colName string) []byte {
	idx, ok := colIndex[colName]
	if !ok {
		return nil
	}
	v := values[idx]
	if v == nil {
		return nil
	}
	switch val := v.(type) {
	case []byte:
		return val
	case string:
		return []byte(val)
	default:
		return []byte(fmt.Sprintf("%v", val))
	}
}
