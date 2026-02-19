# Configuration

wafflex uses a `.wafflex.yaml` config file to centralize parameters. CLI flags override config values.

## Config File Resolution

The config file is searched in this order (first found wins):

1. `--config <path>` flag (explicit)
2. `.wafflex.yaml` in the current directory
3. `~/.wafflex.yaml` in the home directory

## Config File Reference

```yaml
# Path to WAF configuration file (.conf for Coraza, .json for custom rules)
coraza-config: ./coraza-test.conf

# Directory for analysis results (auto-generated timestamped filenames)
results-dir: ./results

# Traffic source configuration
traffic:
  # Reader type: gor, custom, or parquet
  type: parquet

  # For gor/custom: path to traffic file (supports ref+s3:// URIs)
  # file: ./traffic.gor

  # For parquet: DuckDB initialization statements
  duckdb:
    init:
      - "INSTALL httpfs"
      - "LOAD httpfs"
      - "SET s3_endpoint = 's3.fr-par.scw.cloud'"
      - "SET s3_region = 'fr-par'"
      - "SET s3_url_style = 'path'"

  # For parquet: SQL query executed by DuckDB
  query: >
    SELECT * FROM read_parquet('s3://bucket/**/*.parquet', hive_partitioning=true)
    WHERE host = '51.15.207.114'

  # For parquet: column mapping (left = logical field, right = parquet column name)
  columns:
    method: req_method
    path: req_path
    host: req_host
    proto: req_http_version
    headers: req_headers
    body: req_body
    client_ip: client_ip
```

## Fields

| Field | Type | Description |
|---|---|---|
| `coraza-config` | string | Path to WAF config file |
| `results-dir` | string | Directory for analysis result files |
| `traffic.type` | string | Traffic format: `gor`, `custom`, `parquet` |
| `traffic.file` | string | Path to traffic file (gor/custom). Supports `ref+s3://` URIs |
| `traffic.query` | string | DuckDB SQL query (parquet only) |
| `traffic.duckdb.init` | list | DuckDB statements executed before the query |
| `traffic.columns` | map | Column mapping for parquet (see [Parquet](parquet.md)) |

`file` and `query` are mutually exclusive.

## CLI Override

All config values can be overridden by CLI flags:

| Config field | CLI flag |
|---|---|
| `coraza-config` | `--coraza-config` / `-c` |
| `traffic.file` | `--gor-file` / `-g` |
| `results-dir` | `--output` / `-o` (explicit path) |

## Environment Variables

DuckDB init statements support `${VAR}` expansion via Go's `os.ExpandEnv`:

```yaml
duckdb:
  init:
    - "SET s3_access_key_id = '${MY_KEY}'"
```

However, DuckDB natively reads standard AWS environment variables — so in most cases you don't need to set credentials in the config at all. See [Parquet - Authentication](parquet.md#authentication).
