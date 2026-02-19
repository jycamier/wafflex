# Parquet Reader Design

## Summary

Add Parquet as a traffic source format using DuckDB as an embedded query engine. Users provide a SQL query in the config that selects from parquet files (local or S3 with Hive partitioning). Column mapping is configurable.

## Config Format

```yaml
traffic:
  type: parquet
  query: >
    SELECT * FROM read_parquet('s3://meshcap-storage/**/*.parquet', hive_partitioning=true)
    WHERE host = '51.15.207.114'
      AND year = 2026 AND month = 3 AND day = 5
  columns:
    method: req_method
    path: req_path
    host: req_host
    proto: req_http_version
    headers: req_headers
    body: req_body
    client_ip: client_ip
```

`file` and `query` are mutually exclusive. `file` is for gor/custom, `query` is for parquet.

## Column Mapping

Configurable via `columns`. Left side = fixed logical fields, right side = parquet column names.

| Logical field | Description | Required |
|---|---|---|
| `method` | HTTP method | yes |
| `path` | Request path/URI | yes |
| `host` | Host header | no |
| `proto` | HTTP version | no |
| `headers` | JSON string of headers map | no |
| `body` | Request body (BLOB or VARCHAR) | no |
| `client_ip` | Client IP address | no |

Headers format: JSON `map[string]string` (e.g. `{"Content-Type":"text/html"}`).

## Architecture

- Use `go-duckdb` CGo driver for embedded DuckDB
- New `ParquetReader` implements `TrafficReader` interface
- Executes the user's SQL query via DuckDB
- Iterates rows, uses column mapping to build `*http.Request`, sends to channel
- DuckDB handles S3 access, Hive partitioning, and filtering natively

## Config Changes

```go
type ColumnMapping struct {
    Method   string `mapstructure:"method"`
    Path     string `mapstructure:"path"`
    Host     string `mapstructure:"host"`
    Proto    string `mapstructure:"proto"`
    Headers  string `mapstructure:"headers"`
    Body     string `mapstructure:"body"`
    ClientIP string `mapstructure:"client_ip"`
}

type TrafficConfig struct {
    Type    string        `mapstructure:"type"`
    File    string        `mapstructure:"file"`
    Query   string        `mapstructure:"query"`
    Columns ColumnMapping `mapstructure:"columns"`
}
```

## File Structure

| File | Role |
|------|------|
| `internal/parser/parquet_reader.go` | ParquetReader implementation |
| `internal/parser/parquet_reader_test.go` | Tests |
| `internal/parser/factory.go` | Add parquet type + query variant |
| `internal/config/config.go` | Add Query + Columns fields |
| `cmd/analyze.go` | Pass query + columns to parser |
