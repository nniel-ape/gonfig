# gonfig

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
        fmt.Fprintln(os.Stderr, err)
        fmt.Fprintln(os.Stderr, gonfig.Usage(&cfg, gonfig.WithEnvPrefix("APP")))
        os.Exit(1)
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
| `gonfig:"key"` | Explicit file/config key (auto-derived if omitted) | `gonfig:"custom_key"` |
| `description:"text"` | Field description for help output | `description:"database host"` |
| `validate:"rules"` | Validation rules (comma-separated) | `validate:"required,min=1"` |

### Name Auto-Derivation

When explicit tags are not provided, names are derived from the struct field path:

| Field Path | Config Key | Env Name | Flag Name |
|-----------|-----------|----------|-----------|
| `Host` | `host` | `HOST` | `--host` |
| `DB.Host` | `db.host` | `DB_HOST` | `--db-host` |
| `LogLevel` | `log_level` | `LOG_LEVEL` | `--log-level` |
| `DB.MaxConn` | `db.max_conn` | `DB_MAX_CONN` | `--db-max-conn` |

With `WithEnvPrefix("APP")`, env names get the prefix: `DB_HOST` → `APP_DB_HOST`.

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
gonfig.WithFile("config.yaml")           // Load from file (format detected by extension)
gonfig.WithEnvPrefix("APP")              // Prefix for env var lookups
gonfig.WithFlags(os.Args[1:])            // Parse CLI flags
gonfig.WithFileContent(data, gonfig.JSON) // Load from bytes (useful for testing/embedding)
```

Multiple `WithFile` calls load files in order; later files override earlier ones.

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

## License

MIT
