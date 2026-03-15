# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Test

```bash
go test ./...                          # run all tests
go test -run TestName ./...            # run a single test
go test -v -count=1 ./...             # verbose, no cache
go test -cover ./...                   # with coverage summary
go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out  # HTML coverage
```

No build step needed — this is a library, not a binary.

## Architecture

Single-package (`gonfig`) library that loads configuration into a Go struct from multiple sources with priority: **flags > env > file > defaults**.

### Loading Pipeline (`gonfig.go:Load`)

`Load` orchestrates 5 sequential steps on a target struct pointer:

1. **Extract fields** (`field.go`) — walks the struct via reflection, collecting `fieldInfo` for each leaf field (name derivation, tags, index path)
2. **Apply defaults** (`defaults.go`) — sets fields from `default` struct tags
3. **Apply file sources** (`file.go`) — decodes JSON/YAML/TOML into `map[string]any`, then walks the map using dot-separated config keys to set fields
4. **Apply env vars** (`env.go`) — looks up `os.LookupEnv` for each field's derived/explicit env name (with optional prefix)
5. **Apply flags** (`flag.go`) — registers all fields as string flags on a `flag.FlagSet`, parses args, and only applies explicitly-set flags
6. **Validate** (`validate.go`) — checks `validate` tag rules (`required`, `min`, `max`, `oneof`), collects all errors into `*ValidationError`

### Key Design Patterns

- **Two type-conversion paths**: `setFieldValue` (string→typed, used by defaults/env/flags) in `value.go` vs `setFieldFromAny` (any→typed, used by file decoders) in `file.go`
- **Name auto-derivation** (`field.go`): `camelToSnake` converts field paths to env names (`DB_HOST`), flag names (`--db-host`), and config keys (`db.host`)
- **Nested struct recursion**: `extractFields` recurses into struct fields (except `time.Duration`), building dot-separated paths and reflect index chains
- **Usage generation** (`usage.go`): builds aligned columnar help text grouped by top-level struct sections
