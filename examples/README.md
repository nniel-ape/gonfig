# gonfig examples

Runnable examples demonstrating gonfig features. Run any example from the repo root:

```bash
go run ./examples/01-basic
```

## Examples

| Example | What it shows |
|---------|--------------|
| [01-basic](01-basic/) | Defaults only — simplest usage |
| [02-config-file](02-config-file/) | Loading from a YAML file |
| [03-all-sources](03-all-sources/) | Full pipeline: defaults + file + env + flags + `--help` |
| [04-validation](04-validation/) | Validation rules and error inspection |
| [05-advanced-types](05-advanced-types/) | Slices, maps, and `time.Duration` |

## Running

All examples use the root module — no per-example `go.mod` needed.

```bash
# Basic defaults
go run ./examples/01-basic

# Config file (run from its directory so it finds config.yaml)
cd examples/02-config-file && go run .

# All sources with env vars and flags
APP_LOG_LEVEL=warn go run ./examples/03-all-sources -- --server-port 9090

# Validation errors
go run ./examples/04-validation

# Advanced types
cd examples/05-advanced-types && go run .
```
