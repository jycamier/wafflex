# Wafflex

<p align="center">
  <img src="docs/img/gopher.png" width="500" alt="Wafflex Gopher" />
</p>

A CLI tool to test WAF (Web Application Firewall) rules by replaying HTTP traffic through Coraza WAF.

## Features

- **Analyze**: Process traffic files through Coraza WAF and export results to JSON
- **Explore**: Interactive TUI to browse and filter blocked requests
- **Diff**: Compare two analysis results to identify changes
- **Multiple traffic sources**: GoReplay `.gor` files, custom format, or Parquet (via DuckDB)
- **Remote files**: Fetch traffic from S3-compatible storage (`ref+s3://`)
- **Config file**: Centralized `.wafflex.yaml` with CLI override support
- **Query cache**: Cache Parquet query results for faster reruns
- **High performance**: Parallel processing with worker pool

## Installation

```bash
go install github.com/jycamier/wafflex@latest
```

Or build from source:

```bash
git clone https://github.com/jycamier/wafflex
cd wafflex
go build -o wafflex
```

## Quick Start

### With Config File

Create `.wafflex.yaml`:

```yaml
coraza-config: ./coraza-test.conf
results-dir: ./results

traffic:
  type: gor
  file: ./traffic.gor
```

```bash
wafflex analyze
wafflex explore results/2026-03-11T14-30-05.json
```

### With CLI Flags

```bash
wafflex analyze -g traffic.gor -c coraza.conf -o results.json
wafflex explore results.json
wafflex diff baseline.json updated.json
```

### With Parquet (S3)

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
    SELECT * FROM read_parquet('s3://bucket/**/*.parquet', hive_partitioning=true)
    WHERE host = '51.15.207.114'
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

## Commands

| Command | Description |
|---|---|
| `analyze` | Process traffic through WAF, export results to JSON |
| `explore <file>` | Interactive TUI to browse blocked requests |
| `diff <file1> <file2>` | Compare two analysis results side-by-side |
| `cache list` | List cached Parquet query results |
| `cache clear` | Delete all cached files |
| `cache dir` | Print cache directory path |

## Documentation

- [Configuration](docs/configuration.md) — Config file format, resolution order, CLI overrides
- [Parquet traffic source](docs/parquet.md) — DuckDB queries, column mapping, S3 auth, Hive partitioning
- [Remote file fetching](docs/remote-files.md) — `ref+s3://` URI scheme, adding providers
- [Query cache](docs/cache.md) — How caching works, CLI commands, cache directory

## Output Format

```json
{
  "metadata": {
    "timestamp": "2026-03-11T14:30:00Z",
    "total_requests": 1000,
    "blocked_requests": 42
  },
  "results": [
    {
      "id": "uuid",
      "request": {
        "method": "GET",
        "url": "/test?param=<script>alert(1)</script>",
        "headers": {"Host": "example.com"}
      },
      "blocked": true,
      "rules_triggered": [
        {"id": "1", "msg": "XSS Attack", "severity": "CRITICAL", "tags": ["attack-xss"]}
      ],
      "interruption": {"action": "deny", "status": 403}
    }
  ]
}
```

## License

Apache 2.0

## Author

Jean-Yves Camier (@jycamier)
