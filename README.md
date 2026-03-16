# Wafflex

<p align="center">
  <img src="docs/img/gopher.png" width="500" alt="Wafflex Gopher" />
</p>

A CLI tool to test WAF (Web Application Firewall) rules by replaying HTTP traffic through Coraza WAF.

## Features

- **Analyze**: Process traffic files through Coraza WAF and export results to JSON
- **Explore**: Interactive TUI to browse and filter blocked requests (group by rule/IP/user-agent)
- **Diff**: Compare analysis results against a stable baseline
- **Baseline**: Manage a reference snapshot for iterative rule development
- **Status**: View current analysis, baseline, and cache state at a glance
- **Multiple traffic sources**: GoReplay `.gor` files, custom format, or Parquet (via DuckDB)
- **Remote files**: Fetch traffic from S3-compatible storage (`ref+s3://`)
- **Config file**: Centralized `.wafflex.yaml` with CLI override support
- **Query cache**: Cache Parquet query results with configurable TTL (default 24h)
- **High performance**: Parallel processing with worker pool

## Installation

### Go Install (recommended)

```bash
go install github.com/jycamier/wafflex@latest
```

### Pre-built binaries

Download the latest binary for your platform from the [Releases page](https://github.com/jycamier/wafflex/releases).

Available for Linux (amd64), macOS (arm64), and Windows (amd64).

### Build from source

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
    timestamp: captured_at
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
| `analyze --diff` | Analyze and diff against baseline |
| `explore [file]` | Interactive TUI to browse blocked requests |
| `diff [file1] [file2]` | Compare baseline vs latest (or two explicit files) |
| `baseline` | Set the latest analysis as baseline |
| `baseline set <file>` | Set a specific file as baseline |
| `baseline -` | Swap to the previous baseline |
| `status` | Show current analysis, baseline, and cache info |
| `query` | Display traffic requests without WAF analysis |
| `query --more` | Show full request details (headers, body, timestamp) |
| `cache list` | List cached Parquet query results |
| `cache clear` | Delete all cached files and analysis results |
| `cache dir` | Print cache directory path |

## Baseline Workflow

The baseline system provides a stable reference point for comparing WAF rule changes:

```bash
# 1. Run initial analysis and set it as baseline
wafflex analyze
wafflex baseline

# 2. Modify your WAF rules (coraza-test.conf)
# 3. Analyze and diff against baseline
wafflex analyze --diff

# 4. Iterate on rules — baseline stays stable
# 5. When satisfied, promote the new result
wafflex baseline

# Swap between current and previous baseline
wafflex baseline -

# Check current state
wafflex status
```

## TUI Shortcuts

| Key | Explore | Diff |
|---|---|---|
| `Tab` | Switch focus (list / detail) | Switch focus (list / panel 1 / panel 2) |
| `g` | Group by: rule, IP, user-agent | Group by: type, rule |
| `/` | Filter by text | Filter by text |
| `q` | Quit | Quit |

## Configuration

### Cache TTL

By default, cached query results and analysis results expire after 24 hours. This prevents stale data when queries don't include date filters.

```yaml
# .wafflex.yaml
cache-ttl: "24h"    # default
cache-ttl: "48h"    # keep 2 days
cache-ttl: "0"      # no expiry
```

### Timestamp Column

Map a parquet column to display request capture dates in `query --more`:

```yaml
columns:
  timestamp: captured_at
```

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
