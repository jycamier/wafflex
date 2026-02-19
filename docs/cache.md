# Query Cache

When using the [Parquet traffic source](parquet.md), wafflex caches DuckDB query results locally to avoid re-fetching from remote storage on subsequent runs.

## How It Works

1. The SQL query is hashed (SHA256)
2. On **cache miss**: the query runs, results are saved as a local Parquet file, then read
3. On **cache hit**: the local Parquet file is read directly (no remote access)

Cache invalidation is based on the query text — if the query changes, a new cache entry is created.

## Cache Directory

Default: `~/.wafflex/cache/`

Override via environment variable:

```bash
export WAFFLEX_CACHE_DIR=/tmp/my-cache
```

## CLI Commands

```bash
# List cached queries with hash and size
wafflex cache list

# Print cache directory path
wafflex cache dir

# Delete all cached files
wafflex cache clear
```

### Example Output

```
$ wafflex cache list
[cb4004b7dd82] 12.3 MB
  SELECT * FROM read_parquet('s3://meshcap-storage/**/*.parquet', hive_partitioning=true)
  WHERE host = '51.15.207.114' AND year = 2026 AND month = 3 AND day = 5

[a1b2c3d4e5f6] 4.7 MB
  SELECT * FROM read_parquet('s3://meshcap-storage/**/*.parquet', hive_partitioning=true)
  WHERE host = '10.0.0.1'
```

## Notes

- The cache stores raw query results as Parquet files, not WAF analysis results
- Changing the query (even whitespace) produces a new cache entry
- To force a fresh fetch, run `wafflex cache clear` before `analyze`
- Cache files are never expired automatically — use `cache clear` for manual cleanup
