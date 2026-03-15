# gonfig

[![CI](https://github.com/nniel-ape/gonfig/actions/workflows/ci.yml/badge.svg)](https://github.com/nniel-ape/gonfig/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/nniel-ape/gonfig.svg)](https://pkg.go.dev/github.com/nniel-ape/gonfig)
[![Go Report Card](https://goreportcard.com/badge/github.com/nniel-ape/gonfig)](https://goreportcard.com/report/github.com/nniel-ape/gonfig)
[![Coverage](https://img.shields.io/badge/coverage-94.1%25-brightgreen)](https://github.com/nniel-ape/gonfig)

Multi-source configuration loading for Go. Define your config as a struct with tags, and gonfig populates it from environment variables, command-line flags, config files (JSON, YAML, TOML), and defaults — with a clear priority order.

## Features

- **Multiple sources**: env vars, CLI flags, JSON/YAML/TOML files, defaults
- **Priority order**: flag > env > file > default
- **Tag-based API**: auto-derives env/flag/file keys from struct field names
- **Nested structs**: `DB.Host` → env `DB_HOST`, flag `--db-host`, file key `db.host`
- **Validation**: `required`, `min`/`max`, `oneof` rules
- **Usage/help generation**: formatted help text from struct metadata
- **Slices and maps**: comma-separated env/flag values, native file arrays/maps
- **Zero dependencies** beyond `gopkg.in/yaml.v3` and `github.com/BurntSushi/toml`

## Install

```
go get github.com/nniel-ape/gonfig
```

Requires Go 1.22 or later.

## Quick Start

```go
package main

import (
    "fmt"
    "log"
    "os"

    "github.com/nniel-ape/gonfig"
)

type Config struct {
    DB struct {
        Host string `default:"localhost" description:"database host" validate:"required"`
        Port int    `default:"5432"      description:"database port" validate:"min=1,max=65535"`
    }
    LogLevel string `default:"info"  description:"logging level" validate:"oneof=debug info warn error"`
    Debug    bool   `default:"false" description:"enable debug mode"`
}

func main() {
    var cfg Config
    err := gonfig.Load(&cfg,
        gonfig.WithEnvPrefix("APP"),
        gonfig.WithFile("config.yaml"),
        gonfig.WithFlags(os.Args[1:]),
    )
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Connecting to %s:%d\n", cfg.DB.Host, cfg.DB.Port)
}
```

## Tag Reference

| Tag | Purpose | Example |
|-----|---------|---------|
| `default:"value"` | Default value | `default:"localhost"` |
| `env:"NAME"` | Explicit env var name (auto-derived if omitted) | `env:"CUSTOM_HOST"` |
| `flag:"name"` | Explicit flag name (auto-derived if omitted) | `flag:"custom-host"` |
| `gonfig:"key"` | Explicit file/config key (auto-derived if omitted). On struct fields, overrides the path segment for all children (see below). | `gonfig:"custom_key"` |
| `description:"text"` | Field description for help output | `description:"database host"` |
| `short:"x"` | Short flag alias (single char, explicit only) | `short:"p"` for `-p` |
| `validate:"rules"` | Validation rules (comma-separated) | `validate:"required,min=1"` |

### Name Auto-Derivation

When explicit tags are not provided, names are derived from the struct field path:

| Field Path | Config Key | Env Name | Flag Name |
|-----------|-----------|----------|-----------|
| `Host` | `host` | `HOST` | `--host` |
| `DB.Host` | `db.host` | `DB_HOST` | `--db-host` |
| `LogLevel` | `log_level` | `LOG_LEVEL` | `--log-level` |
| `DB.MaxConn` | `db.max_conn` | `DB_MAX_CONN` | `--db-max-conn` |
| `APIURL` | `api_url` | `API_URL` | `--api-url` |
| `MarketIDs` | `market_ids` | `MARKET_IDS` | `--market-ids` |

Common acronyms (`API`, `URL`, `HTTP`, `HTTPS`, `ID`, `IP`, `RPC`, `SQL`, `URI`, `DNS`, `SSH`, `SSL`, `TLS`, `TCP`, `UDP`) are recognized and split correctly.

With `WithEnvPrefix("APP")`, env names get the prefix: `DB_HOST` → `APP_DB_HOST`.

### Struct-Level `gonfig` Tag

The `gonfig` tag on a **struct field** overrides the path segment used by all children. This is useful when the Go struct name doesn't match the desired config key:

```go
type Config struct {
    Strategy Strategy `gonfig:"latemomentum"`
}

type Strategy struct {
    Name   string
    Weight float64
}
// Strategy.Name → env LATEMOMENTUM_NAME, flag --latemomentum-name, config key latemomentum.name
// Strategy.Weight → env LATEMOMENTUM_WEIGHT, flag --latemomentum-weight, config key latemomentum.weight
```

On **leaf fields**, the `gonfig` tag only overrides the config file key (env and flag names are unaffected).

## Config File Formats

All formats map to the same struct fields via config keys:

### YAML

```yaml
db:
  host: myhost
  port: 3306
log_level: debug
```

### TOML

```toml
log_level = "debug"

[db]
host = "myhost"
port = 3306
```

### JSON

```json
{
  "db": {"host": "myhost", "port": 3306},
  "log_level": "debug"
}
```

## Priority Order

Sources are applied lowest-to-highest priority:

1. **Defaults** — `default` struct tag values
2. **File** — config file values (JSON, YAML, TOML)
3. **Environment** — env var values
4. **Flags** — command-line flag values

Each source only overrides fields it explicitly sets. Unset fields retain values from lower-priority sources.

## API Reference

### Load

```go
func Load(target any, opts ...Option) error
```

Populates the target struct from all configured sources. The target must be a non-nil pointer to a struct.

### Options

```go
gonfig.WithFile("config.yaml")            // Load from file (format detected by extension)
gonfig.WithEnvPrefix("APP")               // Prefix for env var lookups
gonfig.WithFlags(os.Args[1:])             // Parse CLI flags
gonfig.WithFileContent(data, gonfig.JSON)  // Load from bytes (useful for testing/embedding)
gonfig.WithAutoHelp(false)                 // Disable auto --help (returns flag.ErrHelp instead)
gonfig.WithoutValidation()                 // Skip validation step (for custom validation)
```

Multiple `WithFile` calls load files in order; later files override earlier ones.

By default, when `WithFlags` is used and `--help`/`-h` is passed, Load prints usage and exits. Use `WithAutoHelp(false)` to receive `flag.ErrHelp` from Load for manual handling.

### Usage

```go
func Usage(target any, opts ...Option) string
```

Generates formatted help text showing flag names, env var names, types, defaults, and descriptions. Fields are grouped by nested struct sections.

### Validation Rules

Rules are specified in the `validate` tag, comma-separated:

| Rule | Applies to | Example |
|------|-----------|---------|
| `required` | All types | `validate:"required"` |
| `min=N` | Numeric types | `validate:"min=1"` |
| `max=N` | Numeric types | `validate:"max=65535"` |
| `oneof=a b c` | All types (space-separated values) | `validate:"oneof=debug info warn error"` |

All validation errors are collected and returned together in a `ValidationError`.

### Error Types

```go
gonfig.ErrInvalidTarget // target is not a non-nil pointer to struct
gonfig.ErrFileNotFound  // specified config file does not exist
gonfig.ErrParse         // type conversion failed
gonfig.ErrValidation    // validation failed (unwrap to *ValidationError for details)
```

Use `errors.Is` and `errors.As` for matching:

```go
var ve *gonfig.ValidationError
if errors.As(err, &ve) {
    for _, fe := range ve.Errors {
        fmt.Printf("  %s: %s\n", fe.Field, fe.Message)
    }
}
```

### Supported Types

| Go Type | Env/Flag (string) | File (native) |
|---------|-------------------|---------------|
| `string` | as-is | `string` |
| `int`, `int64` | parsed | `float64` → int |
| `float64` | parsed | `float64` |
| `bool` | parsed | `bool` |
| `time.Duration` | `time.ParseDuration` | `string` → parse |
| `[]string` | comma-separated | native array |
| `[]int` | comma-separated | native array |
| `[]float64` | comma-separated | native array |
| `map[string]string` | not supported | native map |
| `map[string]any` | not supported | native map |

## Examples

See the [examples/](examples/) directory for runnable demos:

| Example | What it shows |
|---------|--------------|
| [01-basic](examples/01-basic/) | Defaults only — simplest usage |
| [02-config-file](examples/02-config-file/) | Loading from a YAML file |
| [03-all-sources](examples/03-all-sources/) | Full pipeline: defaults + file + env + flags + auto `--help` |
| [04-validation](examples/04-validation/) | Validation rules and error inspection |
| [05-advanced-types](examples/05-advanced-types/) | Slices, maps, and `time.Duration` |
| [06-manual-handling](examples/06-manual-handling/) | Manual `--help` and validation error handling |

## License

MIT
