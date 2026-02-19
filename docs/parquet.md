# Parquet Traffic Source

wafflex can read HTTP traffic from Parquet files using an embedded DuckDB engine. This enables querying remote data (S3, GCS) with SQL filtering, Hive partitioning, and local caching.

## Quick Start

```yaml
# .wafflex.yaml
coraza-config: ./coraza-test.conf
results-dir: ./results

traffic:
  type: parquet
  duckdb:
    init:
      - "INSTALL httpfs"
      - "LOAD httpfs"
      - "SET s3_endpoint = 's3.fr-par.scw.cloud'"
      - "SET s3_region = 'fr-par'"
      - "SET s3_url_style = 'path'"
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

```bash
export AWS_ACCESS_KEY_ID=your-key
export AWS_SECRET_ACCESS_KEY=your-secret
wafflex analyze
```

## Column Mapping

The `columns` section maps logical HTTP fields to your Parquet column names. This makes the reader schema-agnostic.

| Logical field | Description | Required |
|---|---|---|
| `method` | HTTP method (GET, POST, ...) | yes |
| `path` | Request path/URI | yes |
| `host` | Host header | no |
| `proto` | HTTP version (e.g. HTTP/1.1) | no |
| `headers` | JSON string of headers map | no |
| `body` | Request body (BLOB or VARCHAR) | no |
| `client_ip` | Client IP address | no |

### Headers Format

Headers must be stored as a JSON-encoded `map[string]string`:

```json
{"Content-Type": "application/json", "Accept": "*/*"}
```

## DuckDB Init Statements

The `duckdb.init` list contains SQL statements executed before the query. Typical uses:

- Install and load extensions (`httpfs`, `aws`)
- Configure S3-compatible endpoints
- Set DuckDB parameters

Statements support `${VAR}` environment variable expansion.

## Authentication

DuckDB natively reads AWS-standard environment variables:

| DuckDB setting | Environment variable |
|---|---|
| `s3_access_key_id` | `AWS_ACCESS_KEY_ID` |
| `s3_secret_access_key` | `AWS_SECRET_ACCESS_KEY` |
| `s3_session_token` | `AWS_SESSION_TOKEN` |
| `s3_region` | `AWS_REGION` or `AWS_DEFAULT_REGION` |
| `s3_endpoint` | `DUCKDB_S3_ENDPOINT` |

For S3-compatible providers (Scaleway, MinIO, etc.), you still need `SET s3_endpoint` and `SET s3_url_style` in `duckdb.init` since those are provider-specific.

### Example: Scaleway

```yaml
duckdb:
  init:
    - "INSTALL httpfs"
    - "LOAD httpfs"
    - "SET s3_endpoint = 's3.fr-par.scw.cloud'"
    - "SET s3_region = 'fr-par'"
    - "SET s3_url_style = 'path'"
```

```bash
export AWS_ACCESS_KEY_ID=SCW...
export AWS_SECRET_ACCESS_KEY=...
```

### Example: AWS S3 (standard)

```yaml
duckdb:
  init:
    - "INSTALL httpfs"
    - "LOAD httpfs"
```

Credentials and region are read automatically from `~/.aws/credentials`, environment variables, or instance profiles.

## Hive Partitioning

If your Parquet files are organized in a Hive-style directory structure:

```
s3://bucket/host=10.0.0.1/year=2026/month=03/day=05/hour=14/data.parquet
```

Use `hive_partitioning=true` in your query:

```sql
SELECT * FROM read_parquet('s3://bucket/**/*.parquet', hive_partitioning=true)
WHERE host = '10.0.0.1' AND year = 2026 AND month = 3
```

DuckDB uses partition columns for predicate pushdown — only matching directories are scanned.

## Query Examples

```sql
-- All traffic for a specific host and day
SELECT * FROM read_parquet('s3://bucket/**/*.parquet', hive_partitioning=true)
WHERE host = '51.15.207.114' AND year = 2026 AND month = 3 AND day = 5

-- Exclude health checks
SELECT * FROM read_parquet('s3://bucket/**/*.parquet', hive_partitioning=true)
WHERE req_path NOT LIKE '%healthz%'
  AND req_path NOT LIKE '%readyz%'
  AND req_headers NOT LIKE '%kube-probe%'

-- Time window (if you have a timestamp column)
SELECT * FROM read_parquet('s3://bucket/**/*.parquet', hive_partitioning=true)
WHERE captured_at BETWEEN '2026-03-10T14:35:00' AND '2026-03-10T14:45:00'

-- Local parquet file
SELECT * FROM read_parquet('./data/traffic.parquet')
```
