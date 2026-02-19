# Diff Workflow Design

## Problem

The current workflow requires too many manual steps:
1. Run `analyze` → get a JSON file path
2. Copy-paste path into `explore <file>` or `diff <file1> <file2>`
3. For diff, manually remember which two files to compare

## Design

### Result File Naming

Replace timestamp-based naming with hash-based naming:

```
results/<hash-query>-<hash-rules>.json
```

- **hash-query** (12 chars): SHA256 of the traffic source identity
  - Parquet: hash of the SQL query string
  - GOR/Custom: hash of the file content
- **hash-rules** (12 chars): SHA256 of the Coraza `.conf` file content

Example: `results/cb4004b7dd82-a1b2c3d4e5f6.json`

If a file with the same name already exists (same traffic + same rules), **skip the analysis entirely** and go straight to the TUI. The result would be identical — no need to recompute.

### `analyze` Default Behavior

```
wafflex analyze
# → computes hash-query + hash-rules
# → if results/cb4004b7dd82-a1b2c3d4e5f6.json exists:
#      → skips analysis, logs "results already exist, skipping analysis"
# → else:
#      → runs WAF analysis
#      → writes results/cb4004b7dd82-a1b2c3d4e5f6.json
# → opens explore TUI on the results
```

### `analyze --diff`

After analysis, find the previous result with the same hash-query but a different hash-rules, and launch the diff TUI:

```
wafflex analyze --diff
# → runs WAF analysis
# → writes results/cb4004b7dd82-a1b2c3d4e5f6.json
# → finds results/cb4004b7dd82-*.json with different hash-rules
# → opens diff TUI comparing old vs new
# → if no previous result found, falls back to explore TUI with a warning
```

Selection logic for the "previous" result: most recent file (by mtime) matching the same hash-query prefix but with a different hash-rules suffix.

### `diff` Without Arguments

Reads `.wafflex.yaml`, computes the current hash-query, finds the two most recent results with that hash-query but different hash-rules:

```
wafflex diff
# → reads config, computes hash-query from current traffic source
# → finds the two most recent results/<hash-query>-*.json with different hash-rules
# → opens diff TUI
# → error if fewer than 2 matching results found
```

### `diff <file1> <file2>` (Unchanged)

Explicit mode remains for comparing any two result files manually.

### `explore` and `explore <file>` (Unchanged)

Standalone explore remains for revisiting past results.

### Hash Computation

Reuse existing SHA256 logic from `internal/cache/cache.go`. Create a shared `internal/hash` package (or add to an existing utility package) with:

```go
func TrafficSourceHash(cfg *config.Config) (string, error)
func RulesHash(corazaConfigPath string) (string, error)
```

`TrafficSourceHash` dispatches based on traffic type:
- `parquet` → SHA256 of the query string
- `gor`/`custom` → SHA256 of the file content

`RulesHash` reads and hashes the Coraza config file content.

Both return the first 12 hex characters of the SHA256.

### Metadata in JSON

Store the hashes in the result metadata for traceability:

```json
{
  "metadata": {
    "timestamp": "2026-03-11T14:30:00Z",
    "total_requests": 1000,
    "blocked_requests": 42,
    "traffic_hash": "cb4004b7dd82",
    "rules_hash": "a1b2c3d4e5f6"
  },
  "results": [...]
}
```

## Commands Summary

| Command | Behavior |
|---|---|
| `analyze` | Analyze + explore TUI |
| `analyze --diff` | Analyze + diff with previous (same traffic, different rules) |
| `analyze -o file.json` | Analyze + write to explicit path (no TUI, backward compat) |
| `diff` | Auto-diff: two most recent results for current traffic source |
| `diff <f1> <f2>` | Explicit diff (unchanged) |
| `explore` | Explore most recent result for current traffic source |
| `explore <file>` | Explore explicit file (unchanged) |

## Out of Scope

- Caching the hash of large GOR files (optimize later if needed)
- Migration of existing timestamp-named result files
