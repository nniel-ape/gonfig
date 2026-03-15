# gonfig: multi-source config library

## Overview
- Go library for loading configuration from multiple sources (env, flags, YAML, TOML, JSON) into a struct using Go tags
- Unified tag-based API: struct field names auto-derive env var names, flag names, and file keys — with explicit override tags when needed
- Priority order: **flag > env > file > default**
- Architecture: **hybrid** — native decoders for files (decode → `map[string]any` → struct mapping), reflection-based overlay for env/flag/default
- Features: nested structs, slices/maps, validation, help/usage generation with `description` tag, env prefix
- Public library at `github.com/nniel-ape/gonfig`

### Tag Design

```go
type Config struct {
    DB struct {
        Host     string `default:"localhost" description:"database host" validate:"required"`
        Port     int    `default:"5432"      description:"database port" validate:"min=1,max=65535"`
        Password string `default:""          description:"database password"`
    }
    LogLevel string `default:"info" description:"logging level" validate:"oneof=debug,info,warn,error"`
    Debug    bool   `default:"false" description:"enable debug mode"`
}

var cfg Config
err := gonfig.Load(&cfg,
    gonfig.WithEnvPrefix("APP"),
    gonfig.WithFile("config.yaml"),
    gonfig.WithFlags(os.Args[1:]),
)
```

**Auto-derivation from field names** (when explicit tags not provided):
- `DB.Host` → env: `APP_DB_HOST`, flag: `--db-host`, file key: `db.host`
- Explicit overrides: `env:"CUSTOM_NAME"`, `flag:"custom-flag"`, `gonfig:"custom_key"`

**Supported tags:**
- `default:"value"` — default value
- `env:"NAME"` — explicit env var name (auto-derived if omitted)
- `flag:"name"` — explicit flag name (auto-derived if omitted)
- `gonfig:"key"` — explicit file/config key (auto-derived from field name if omitted)
- `description:"text"` — field description for help/usage output
- `validate:"rules"` — validation rules (required, min, max, oneof, etc.)

### File Format Examples

All formats map to the same struct:

```yaml
# config.yaml
db:
  host: myhost
  port: 3306
log_level: debug
```

```toml
# config.toml
log_level = "debug"
[db]
host = "myhost"
port = 3306
```

```json
{
  "db": {"host": "myhost", "port": 3306},
  "log_level": "debug"
}
```

## Context (from discovery)
- Files/components involved: greenfield project, no existing code
- Module path: `github.com/nniel-ape/gonfig`
- Go version: 1.22 (minimum — maximizes compatibility for consumers)
- Dependencies: `gopkg.in/yaml.v3`, `github.com/BurntSushi/toml`, standard library for JSON/flags
- No external validation library — implement core validation rules natively to minimize dependencies

## Development Approach
- **Testing approach**: TDD (tests first)
- Complete each task fully before moving to the next
- Make small, focused changes
- **CRITICAL: every task MUST include new/updated tests** for code changes in that task
  - tests are not optional — they are a required part of the checklist
  - write unit tests for new functions/methods
  - write unit tests for modified functions/methods
  - add new test cases for new code paths
  - update existing test cases if behavior changes
  - tests cover both success and error scenarios
- **CRITICAL: all tests must pass before starting next task** — no exceptions
- **CRITICAL: update this plan file when scope changes during implementation**
- Run tests after each change
- Maintain backward compatibility

## Testing Strategy
- **Unit tests**: required for every task (see Development Approach above)
- **Table-driven tests**: use Go idiomatic table-driven test patterns throughout
- **Testdata files**: store fixture config files in `testdata/` directories
- **No external test dependencies**: use only standard library `testing` package

## Progress Tracking
- Mark completed items with `[x]` immediately when done
- Add newly discovered tasks with ➕ prefix
- Document issues/blockers with ⚠️ prefix
- Update plan if implementation deviates from original scope
- Keep plan in sync with actual work done

## Implementation Steps

### Task 1: Project scaffolding
- [x] run `go mod init github.com/nniel-ape/gonfig` and set Go 1.22
- [x] create directory structure: root package, `testdata/`
- [x] create initial `gonfig.go` with package doc comment and `Load()` stub that returns nil
- [x] create `gonfig_test.go` with a placeholder test that calls `Load()` and asserts no error
- [x] run tests — must pass before next task

### Task 2: Struct field info extractor (reflection core)
- [x] define `fieldInfo` struct: `Name`, `Path` (dot-separated), `Type`, `DefaultVal`, `EnvName`, `FlagName`, `ConfigKey`, `Description`, `ValidateRules`, `Index` path for reflect access
- [x] implement `extractFields(v reflect.Value, prefix string) []fieldInfo` — recursively walks struct, extracts tag values, auto-derives env/flag/config names from field path
- [x] implement name derivation helpers: `toEnvName(path string) string` (DB.Host → DB_HOST), `toFlagName(path string) string` (DB.Host → db-host), `toConfigKey(path string) string` (DB.Host → db.host)
- [x] write tests for `extractFields` with flat struct (multiple field types: string, int, bool, float)
- [x] write tests for `extractFields` with nested struct (verify dot-path generation)
- [x] write tests for `extractFields` with explicit tag overrides (env, flag, gonfig tags)
- [x] write tests for name derivation helpers (camelCase, multi-word, acronyms)
- [x] run tests — must pass before next task

### Task 3: Default provider
- [x] implement `applyDefaults(target any, fields []fieldInfo) error` — sets field values from `default` tag
- [x] implement type conversion: `setFieldValue(field reflect.Value, raw string) error` for string, int, int64, float64, bool, time.Duration, `[]string`, `[]int`
- [x] write tests for `applyDefaults` — string, int, bool, float, duration defaults
- [x] write tests for `applyDefaults` — missing default tag (field unchanged)
- [x] write tests for `applyDefaults` — invalid default value for type (expect error)
- [x] write tests for `setFieldValue` — all supported types including edge cases (empty string, zero, negative)
- [x] run tests — must pass before next task

### Task 4: File provider — JSON
- [x] implement `loadFile(target any, path string, fields []fieldInfo) error` — detect format from extension, decode into `map[string]any`, then map onto struct fields
- [x] implement `decodeJSON(r io.Reader) (map[string]any, error)` using `encoding/json`
- [x] implement `applyMap(target any, data map[string]any, fields []fieldInfo) error` — walk flat/nested map, match to fields by config key, set values
- [x] create `testdata/valid.json`, `testdata/nested.json`, `testdata/empty.json`, `testdata/invalid.json` fixtures
- [x] write tests for JSON loading — flat config
- [x] write tests for JSON loading — nested config
- [x] write tests for JSON loading — file not found, invalid JSON (error cases)
- [x] run tests — must pass before next task

### Task 5: File provider — YAML
- [x] add `gopkg.in/yaml.v3` dependency
- [x] implement `decodeYAML(r io.Reader) (map[string]any, error)`
- [x] create `testdata/valid.yaml`, `testdata/nested.yaml` fixtures
- [x] write tests for YAML loading — flat and nested config
- [x] write tests for YAML loading — invalid YAML (error case)
- [x] run tests — must pass before next task

### Task 6: File provider — TOML
- [x] add `github.com/BurntSushi/toml` dependency
- [x] implement `decodeTOML(r io.Reader) (map[string]any, error)`
- [x] create `testdata/valid.toml`, `testdata/nested.toml` fixtures
- [x] write tests for TOML loading — flat and nested config
- [x] write tests for TOML loading — invalid TOML (error case)
- [x] run tests — must pass before next task

### Task 7: Env provider
- [x] implement `applyEnv(target any, fields []fieldInfo, prefix string) error` — for each field, check `os.LookupEnv` with optional prefix, set value if found
- [x] write tests for env loading — basic types (string, int, bool) using `t.Setenv`
- [x] write tests for env loading — with prefix (APP_DB_HOST)
- [x] write tests for env loading — env var not set (field unchanged)
- [x] write tests for env loading — invalid env value for type (expect error)
- [x] run tests — must pass before next task

### Task 8: Flag provider
- [x] implement `applyFlags(target any, fields []fieldInfo, args []string) error` — create `flag.FlagSet`, register flags for all fields with descriptions and defaults, parse args, apply only explicitly-set flags
- [x] implement detection of explicitly-set flags (use `FlagSet.Visit`) to distinguish "flag set to zero" from "flag not provided"
- [x] write tests for flag parsing — basic types
- [x] write tests for flag parsing — only explicitly-set flags override (unset flags don't clobber file/env values)
- [x] write tests for flag parsing — unknown flag handling (error case)
- [x] write tests for flag parsing — `--help` triggers ErrHelp
- [x] run tests — must pass before next task

### Task 9: Slice and map support
- [x] extend `setFieldValue` for `[]string` (comma-separated), `[]int`, `[]float64`
- [x] extend `applyMap` for slice fields from file sources (native arrays)
- [x] extend `applyMap` for `map[string]string` and `map[string]any` fields from file sources
- [x] write tests for slice fields — env (comma-separated), flag (comma-separated), file (native array)
- [x] write tests for map fields — file source (native map)
- [x] write tests for edge cases — empty slice, single-element slice, nested slices
- [x] run tests — must pass before next task

### Task 10: Load orchestrator and public API
- [x] define option types: `Option`, `WithEnvPrefix(string)`, `WithFile(string)`, `WithFlags([]string)`, `WithFileContent([]byte, Format)`
- [x] implement `Load(target any, opts ...Option) error` — validate target is pointer-to-struct, extract fields, then apply in order: defaults → file → env → flags
- [x] define exported error types: `ErrInvalidTarget`, `ErrFileNotFound`, `ErrParse`, `ErrValidation`
- [x] write tests for `Load` end-to-end — all sources combined, verify priority order
- [x] write tests for `Load` — only defaults (no file, no env, no flags)
- [x] write tests for `Load` — file + env override
- [x] write tests for `Load` — flag overrides everything
- [x] write tests for `Load` — error cases (nil target, non-pointer, non-struct)
- [x] run tests — must pass before next task

### Task 11: Validation
- [x] implement `validate(target any, fields []fieldInfo) error` — run after all sources applied
- [x] implement validation rules: `required` (non-zero value), `min=N`/`max=N` (numeric range), `oneof=a,b,c` (allowed values)
- [x] collect all validation errors into a single `ValidationError` with per-field details (don't fail on first)
- [x] write tests for `required` — zero value fails, non-zero passes
- [x] write tests for `min`/`max` — int and float boundaries, out-of-range errors
- [x] write tests for `oneof` — valid value passes, invalid fails
- [x] write tests for combined rules — field with `required,min=1,max=100`
- [x] write tests for `ValidationError` — multiple fields fail, error message lists all
- [x] run tests — must pass before next task

### Task 12: Help/usage generation
- [x] implement `Usage(target any, opts ...Option) string` — generates formatted usage text from struct metadata
- [x] format: flag name, env var name, type, default, description — aligned columns
- [x] group by nested struct sections (e.g., "DB:" header)
- [x] write tests for `Usage` — flat struct output matches expected text
- [x] write tests for `Usage` — nested struct with section headers
- [x] write tests for `Usage` — fields without description tag (still listed, no description)
- [x] run tests — must pass before next task

### Task 13: Verify acceptance criteria
- [ ] verify all formats work: JSON, YAML, TOML, env, flags — each tested in isolation and combined
- [ ] verify priority order: flag > env > file > default — tested with all sources setting same key
- [ ] verify nested structs work across all sources
- [ ] verify slices and maps work across applicable sources
- [ ] verify validation catches invalid config and reports all errors
- [ ] verify help/usage output is correct and complete
- [ ] run full test suite
- [ ] run linter (`golangci-lint run` if configured, otherwise `go vet ./...`)
- [ ] verify test coverage meets 80%+ (`go test -coverprofile=coverage.out ./... && go tool cover -func=coverage.out`)

### Task 14: [Final] Documentation
- [ ] write README.md with: badges, install instructions, quick start, full tag reference, all format examples, API reference, priority explanation
- [ ] add Go doc comments on all exported types, functions, and methods
- [ ] add runnable `Example` test functions for godoc
- [ ] update this plan with any deviations

## Technical Details

### Package Structure
```
gonfig/
├── gonfig.go           # Load(), Option types, public API
├── gonfig_test.go      # Integration/end-to-end tests
├── field.go            # fieldInfo, extractFields, name derivation
├── field_test.go
├── value.go            # setFieldValue, type conversion
├── value_test.go
├── defaults.go         # applyDefaults
├── defaults_test.go
├── file.go             # loadFile, decodeJSON/YAML/TOML, applyMap
├── file_test.go
├── env.go              # applyEnv
├── env_test.go
├── flag.go             # applyFlags
├── flag_test.go
├── validate.go         # validate, ValidationError
├── validate_test.go
├── usage.go            # Usage()
├── usage_test.go
├── testdata/
│   ├── valid.json
│   ├── valid.yaml
│   ├── valid.toml
│   ├── nested.json
│   ├── nested.yaml
│   ├── nested.toml
│   ├── empty.json
│   └── invalid.json
├── go.mod
├── go.sum
└── README.md
```

### Type Conversion Matrix

| Go Type        | Env/Flag (string) | File (any)           |
|----------------|-------------------|----------------------|
| `string`       | as-is             | `string`             |
| `int`, `int64` | `strconv.Atoi`    | `float64` → int cast |
| `float64`      | `strconv.ParseFloat` | `float64`         |
| `bool`         | `strconv.ParseBool` | `bool`             |
| `time.Duration`| `time.ParseDuration` | `string` → parse  |
| `[]string`     | comma-split       | `[]any` → strings    |
| `[]int`        | comma-split+atoi  | `[]any` → ints       |
| `map[string]string` | not supported | `map[string]any`     |

### Name Auto-Derivation Rules

| Field Path   | Config Key   | Env Name    | Flag Name   |
|-------------|-------------|-------------|-------------|
| `Host`      | `host`      | `HOST`      | `--host`    |
| `DB.Host`   | `db.host`   | `DB_HOST`   | `--db-host` |
| `LogLevel`  | `log_level` | `LOG_LEVEL` | `--log-level` |
| `DB.MaxConn`| `db.max_conn`| `DB_MAX_CONN`| `--db-max-conn` |

CamelCase → snake_case for config keys and env names, kebab-case for flags.

### Error Types
- `ErrInvalidTarget` — Load called with non-pointer or non-struct
- `ErrFileNotFound` — specified config file doesn't exist
- `ErrParse` — type conversion failed (wraps source info: which field, which source, what value)
- `ErrValidation` — validation failed (wraps `[]FieldError` with field name, rule, message)

## Post-Completion

**Manual verification:**
- Test with a real CLI app consuming gonfig
- Verify `go doc` output looks clean on pkg.go.dev preview
- Test cross-platform env var behavior (case sensitivity on Linux vs macOS)

**Publishing:**
- Tag `v0.1.0` release
- Push to GitHub at `github.com/nniel-ape/gonfig`
- Verify module is discoverable on pkg.go.dev
