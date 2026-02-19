# Config File Design

## Summary

Add a `.wafflex.yaml` configuration file to centralize parameters for the `analyze` command. The file provides default values that CLI flags can override.

## Config File Format

```yaml
# Configuration WAF
coraza-config: ./coraza-test.conf

# Source de trafic
traffic:
  type: gor
  file: ./traffic.gor

# Dossier de stockage des résultats
results-dir: ./results
```

### Fields

| Field | Type | Description |
|-------|------|-------------|
| `coraza-config` | string | Path to WAF config file (.conf, .json, .yaml) |
| `traffic.type` | string | Traffic format: `gor`, `custom` |
| `traffic.file` | string | Path to traffic capture file |
| `results-dir` | string | Directory where analysis results are stored |

## Config Resolution Order

Priority (highest to lowest):

1. `--config <path>` CLI flag (explicit path)
2. `.wafflex.yaml` in current working directory
3. `~/.wafflex.yaml` in user home directory

## CLI Override Behavior

CLI flags override config file values. Config file provides defaults.

| Config field | CLI flag that overrides it |
|-------------|--------------------------|
| `coraza-config` | `--coraza-config` / `-c` |
| `traffic.file` | `--gor-file` / `-g` |
| `results-dir` | `--output` / `-o` (when explicit path given) |

## Analyze Command Changes

- All flags become optional when config file exists
- When `--output` is not specified, results are written to `results-dir/` with auto-generated filename: `YYYY-MM-DDTHH-MM-SS.json`
- The `results-dir` directory is created automatically if it doesn't exist
- `traffic.type` feeds the parser factory directly (no extension guessing needed)

## Commands Not Changed

- `explore`: keeps positional argument
- `diff`: no changes (user still deciding)

## Implementation

- Use **Viper** library (integrates with Cobra) for config file loading and flag binding
- Add `--config` persistent flag on root command
- Add `internal/config/` package for config struct and loading logic
