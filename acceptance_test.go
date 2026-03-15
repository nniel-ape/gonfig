package gonfig

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Acceptance tests for Task 13: verify all acceptance criteria end-to-end.

// --- Criterion 1: All formats work in isolation and combined ---

func TestAcceptance_AllFormats_Isolation(t *testing.T) {
	type cfg struct {
		DB struct {
			Host string `default:"localhost"`
			Port int    `default:"5432"`
		}
		LogLevel string `default:"info"`
		Debug    bool   `default:"false"`
	}

	formats := []struct {
		name string
		file string
	}{
		{"json", "testdata/nested.json"},
		{"yaml", "testdata/nested.yaml"},
		{"toml", "testdata/nested.toml"},
	}

	for _, f := range formats {
		t.Run(f.name, func(t *testing.T) {
			var c cfg
			err := Load(&c, WithFile(f.file))
			if err != nil {
				t.Fatalf("Load(%s) error: %v", f.name, err)
			}
			if c.DB.Host != "dbhost" {
				t.Errorf("DB.Host = %q, want %q", c.DB.Host, "dbhost")
			}
			if c.DB.Port != 5432 {
				t.Errorf("DB.Port = %d, want %d", c.DB.Port, 5432)
			}
			if c.LogLevel != "warn" {
				t.Errorf("LogLevel = %q, want %q", c.LogLevel, "warn")
			}
		})
	}
}

func TestAcceptance_AllFormats_EnvOverridesFile(t *testing.T) {
	type cfg struct {
		DB struct {
			Host string `default:"localhost"`
			Port int    `default:"5432"`
		}
		LogLevel string `default:"info"`
	}

	formats := []struct {
		name string
		file string
	}{
		{"json", "testdata/nested.json"},
		{"yaml", "testdata/nested.yaml"},
		{"toml", "testdata/nested.toml"},
	}

	for _, f := range formats {
		t.Run(f.name, func(t *testing.T) {
			t.Setenv("TEST_DB_HOST", "envhost")

			var c cfg
			err := Load(&c, WithFile(f.file), WithEnvPrefix("TEST"))
			if err != nil {
				t.Fatalf("Load error: %v", err)
			}
			if c.DB.Host != "envhost" {
				t.Errorf("DB.Host = %q, want %q (env should override %s file)", c.DB.Host, "envhost", f.name)
			}
			// File value should remain for non-overridden fields.
			if c.LogLevel != "warn" {
				t.Errorf("LogLevel = %q, want %q (from file)", c.LogLevel, "warn")
			}
		})
	}
}

func TestAcceptance_AllFormats_FlagOverridesAll(t *testing.T) {
	type cfg struct {
		DB struct {
			Host string `default:"localhost"`
			Port int    `default:"5432"`
		}
		LogLevel string `default:"info"`
	}

	formats := []struct {
		name string
		file string
	}{
		{"json", "testdata/nested.json"},
		{"yaml", "testdata/nested.yaml"},
		{"toml", "testdata/nested.toml"},
	}

	for _, f := range formats {
		t.Run(f.name, func(t *testing.T) {
			t.Setenv("ACC_LOG_LEVEL", "error")

			var c cfg
			err := Load(&c,
				WithFile(f.file),
				WithEnvPrefix("ACC"),
				WithFlags([]string{"--log-level", "trace", "--db-host", "flaghost"}),
			)
			if err != nil {
				t.Fatalf("Load error: %v", err)
			}
			// Flag should override env and file.
			if c.LogLevel != "trace" {
				t.Errorf("LogLevel = %q, want %q (flag should override env)", c.LogLevel, "trace")
			}
			if c.DB.Host != "flaghost" {
				t.Errorf("DB.Host = %q, want %q (flag should override file)", c.DB.Host, "flaghost")
			}
		})
	}
}

// --- Criterion 2: Priority order flag > env > file > default ---

func TestAcceptance_PriorityOrder_AllSourcesSameKey(t *testing.T) {
	type cfg struct {
		Value string `default:"from-default"`
	}

	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config.json")
	if err := os.WriteFile(cfgFile, []byte(`{"value":"from-file"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	// Test 1: default only
	t.Run("default_only", func(t *testing.T) {
		var c cfg
		if err := Load(&c); err != nil {
			t.Fatal(err)
		}
		if c.Value != "from-default" {
			t.Errorf("Value = %q, want %q", c.Value, "from-default")
		}
	})

	// Test 2: file overrides default
	t.Run("file_overrides_default", func(t *testing.T) {
		var c cfg
		if err := Load(&c, WithFile(cfgFile)); err != nil {
			t.Fatal(err)
		}
		if c.Value != "from-file" {
			t.Errorf("Value = %q, want %q", c.Value, "from-file")
		}
	})

	// Test 3: env overrides file
	t.Run("env_overrides_file", func(t *testing.T) {
		t.Setenv("PRI_VALUE", "from-env")
		var c cfg
		if err := Load(&c, WithFile(cfgFile), WithEnvPrefix("PRI")); err != nil {
			t.Fatal(err)
		}
		if c.Value != "from-env" {
			t.Errorf("Value = %q, want %q", c.Value, "from-env")
		}
	})

	// Test 4: flag overrides env
	t.Run("flag_overrides_env", func(t *testing.T) {
		t.Setenv("PRI2_VALUE", "from-env")
		var c cfg
		if err := Load(&c,
			WithFile(cfgFile),
			WithEnvPrefix("PRI2"),
			WithFlags([]string{"--value", "from-flag"}),
		); err != nil {
			t.Fatal(err)
		}
		if c.Value != "from-flag" {
			t.Errorf("Value = %q, want %q", c.Value, "from-flag")
		}
	})
}

// --- Criterion 3: Nested structs work across all sources ---

func TestAcceptance_NestedStructs_AllSources(t *testing.T) {
	type cfg struct {
		Server struct {
			Host    string `default:"0.0.0.0"`
			Port    int    `default:"8080"`
			Timeout string `default:"30s"`
		}
		DB struct {
			Host     string `default:"localhost"`
			Port     int    `default:"5432"`
			Database string `default:"mydb"`
		}
	}

	dir := t.TempDir()

	// JSON file sets server.host and db.host
	cfgFile := filepath.Join(dir, "config.yaml")
	yamlContent := `
server:
  host: filehost
  port: 9090
db:
  host: filedb
  port: 3306
  database: prod
`
	if err := os.WriteFile(cfgFile, []byte(yamlContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// Env overrides db.host
	t.Setenv("NEST_DB_HOST", "envdbhost")
	// Env overrides server.timeout
	t.Setenv("NEST_SERVER_TIMEOUT", "60s")

	var c cfg
	err := Load(&c,
		WithFile(cfgFile),
		WithEnvPrefix("NEST"),
		WithFlags([]string{"--server-port", "4443"}),
	)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	// server.host: file=filehost (no env/flag)
	if c.Server.Host != "filehost" {
		t.Errorf("Server.Host = %q, want %q", c.Server.Host, "filehost")
	}
	// server.port: file=9090, flag=4443 -> flag wins
	if c.Server.Port != 4443 {
		t.Errorf("Server.Port = %d, want %d", c.Server.Port, 4443)
	}
	// server.timeout: default=30s, env=60s -> env wins
	if c.Server.Timeout != "60s" {
		t.Errorf("Server.Timeout = %q, want %q", c.Server.Timeout, "60s")
	}
	// db.host: file=filedb, env=envdbhost -> env wins
	if c.DB.Host != "envdbhost" {
		t.Errorf("DB.Host = %q, want %q", c.DB.Host, "envdbhost")
	}
	// db.port: file=3306 (no env/flag)
	if c.DB.Port != 3306 {
		t.Errorf("DB.Port = %d, want %d", c.DB.Port, 3306)
	}
	// db.database: file=prod (no env/flag)
	if c.DB.Database != "prod" {
		t.Errorf("DB.Database = %q, want %q", c.DB.Database, "prod")
	}
}

// --- Criterion 4: Slices and maps work across applicable sources ---

func TestAcceptance_Slices_AllSources(t *testing.T) {
	type cfg struct {
		Tags  []string  `default:"a,b"`
		Ports []int     `default:"80,443"`
		Rates []float64 `default:"1.0,2.0"`
	}

	// Test with file sources (all three formats).
	for _, format := range []struct {
		name string
		file string
	}{
		{"json", "testdata/slices.json"},
		{"yaml", "testdata/slices.yaml"},
		{"toml", "testdata/slices.toml"},
	} {
		t.Run("file_"+format.name, func(t *testing.T) {
			var c cfg
			err := Load(&c, WithFile(format.file))
			if err != nil {
				t.Fatalf("Load(%s) error: %v", format.name, err)
			}
			wantTags := []string{"web", "api", "v2"}
			if len(c.Tags) != len(wantTags) {
				t.Fatalf("Tags = %v, want %v", c.Tags, wantTags)
			}
			for i, tag := range c.Tags {
				if tag != wantTags[i] {
					t.Errorf("Tags[%d] = %q, want %q", i, tag, wantTags[i])
				}
			}
			wantPorts := []int{8080, 8443, 9090}
			if len(c.Ports) != len(wantPorts) {
				t.Fatalf("Ports = %v, want %v", c.Ports, wantPorts)
			}
			for i, p := range c.Ports {
				if p != wantPorts[i] {
					t.Errorf("Ports[%d] = %d, want %d", i, p, wantPorts[i])
				}
			}
		})
	}

	// Test with env (comma-separated).
	t.Run("env_override", func(t *testing.T) {
		t.Setenv("SL_TAGS", "x,y,z")
		t.Setenv("SL_PORTS", "1,2,3")

		var c cfg
		err := Load(&c, WithEnvPrefix("SL"))
		if err != nil {
			t.Fatalf("Load error: %v", err)
		}
		wantTags := []string{"x", "y", "z"}
		if len(c.Tags) != len(wantTags) {
			t.Fatalf("Tags = %v, want %v", c.Tags, wantTags)
		}
		for i, tag := range c.Tags {
			if tag != wantTags[i] {
				t.Errorf("Tags[%d] = %q, want %q", i, tag, wantTags[i])
			}
		}
	})

	// Test with flags (comma-separated).
	t.Run("flag_override", func(t *testing.T) {
		var c cfg
		err := Load(&c, WithFlags([]string{"--tags", "f1,f2", "--ports", "100,200"}))
		if err != nil {
			t.Fatalf("Load error: %v", err)
		}
		wantTags := []string{"f1", "f2"}
		if len(c.Tags) != len(wantTags) {
			t.Fatalf("Tags = %v, want %v", c.Tags, wantTags)
		}
		for i, tag := range c.Tags {
			if tag != wantTags[i] {
				t.Errorf("Tags[%d] = %q, want %q", i, tag, wantTags[i])
			}
		}
	})
}

func TestAcceptance_Maps_FileSources(t *testing.T) {
	type cfg struct {
		Labels   map[string]string `gonfig:"labels"`
		Metadata map[string]any    `gonfig:"metadata"`
	}

	for _, format := range []struct {
		name string
		file string
	}{
		{"json", "testdata/maps.json"},
		{"yaml", "testdata/maps.yaml"},
		{"toml", "testdata/maps.toml"},
	} {
		t.Run(format.name, func(t *testing.T) {
			var c cfg
			err := Load(&c, WithFile(format.file))
			if err != nil {
				t.Fatalf("Load(%s) error: %v", format.name, err)
			}
			if c.Labels["env"] != "production" {
				t.Errorf("Labels[env] = %q, want %q", c.Labels["env"], "production")
			}
			if c.Labels["region"] != "us-east-1" {
				t.Errorf("Labels[region] = %q, want %q", c.Labels["region"], "us-east-1")
			}
			if c.Labels["team"] != "platform" {
				t.Errorf("Labels[team] = %q, want %q", c.Labels["team"], "platform")
			}
			if len(c.Labels) != 3 {
				t.Errorf("len(Labels) = %d, want 3", len(c.Labels))
			}

			// Metadata is map[string]any.
			if c.Metadata == nil {
				t.Fatal("Metadata is nil")
			}
			if v, ok := c.Metadata["version"]; !ok || v != "1.0" {
				t.Errorf("Metadata[version] = %v, want %q", v, "1.0")
			}
		})
	}
}

// --- Criterion 5: Validation catches invalid config and reports all errors ---

func TestAcceptance_Validation_MultipleErrors(t *testing.T) {
	type cfg struct {
		Host     string `validate:"required"`
		Port     int    `validate:"required,min=1,max=65535"`
		LogLevel string `validate:"oneof=debug info warn error"`
	}

	// All fields invalid: Host is empty, Port is 0 (zero value), LogLevel is invalid.
	var c cfg
	c.LogLevel = "invalid"
	err := Load(&c)
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}

	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected *ValidationError, got %T: %v", err, err)
	}

	// Should have errors for Host (required), Port (required + min), LogLevel (oneof).
	if len(ve.Errors) < 3 {
		t.Errorf("expected at least 3 validation errors, got %d: %v", len(ve.Errors), ve.Errors)
	}

	// Check specific fields are reported.
	fieldsSeen := make(map[string]bool)
	for _, fe := range ve.Errors {
		fieldsSeen[fe.Field] = true
	}
	for _, field := range []string{"Host", "Port", "LogLevel"} {
		if !fieldsSeen[field] {
			t.Errorf("expected validation error for field %q, not found in %v", field, ve.Errors)
		}
	}
}

func TestAcceptance_Validation_PassesWithValidConfig(t *testing.T) {
	type cfg struct {
		Host     string `default:"localhost" validate:"required"`
		Port     int    `default:"8080"      validate:"required,min=1,max=65535"`
		LogLevel string `default:"info"      validate:"oneof=debug info warn error"`
	}

	var c cfg
	err := Load(&c)
	if err != nil {
		t.Fatalf("Load with valid defaults should pass validation, got: %v", err)
	}
}

func TestAcceptance_Validation_ErrorMessage_ListsAllFields(t *testing.T) {
	type cfg struct {
		A string `validate:"required"`
		B string `validate:"required"`
		C int    `validate:"min=10"`
	}

	var c cfg
	err := Load(&c)
	if err == nil {
		t.Fatal("expected error")
	}

	msg := err.Error()
	// All three fields should be mentioned.
	for _, field := range []string{"A", "B", "C"} {
		if !strings.Contains(msg, field) {
			t.Errorf("error message %q should mention field %q", msg, field)
		}
	}
}

// --- Criterion 6: Help/usage output is correct and complete ---

func TestAcceptance_Usage_CompleteOutput(t *testing.T) {
	type cfg struct {
		DB struct {
			Host string `default:"localhost" description:"database host"`
			Port int    `default:"5432"      description:"database port"`
		}
		LogLevel string `default:"info"  description:"logging level"`
		Debug    bool   `default:"false" description:"enable debug mode"`
	}

	var c cfg
	output := Usage(&c, WithEnvPrefix("APP"))

	// Verify flag names are present.
	for _, flag := range []string{"--log-level", "--debug", "--db-host", "--db-port"} {
		if !strings.Contains(output, flag) {
			t.Errorf("Usage output should contain flag %q\nGot:\n%s", flag, output)
		}
	}

	// Verify env var names with prefix.
	for _, env := range []string{"APP_LOG_LEVEL", "APP_DEBUG", "APP_DB_HOST", "APP_DB_PORT"} {
		if !strings.Contains(output, env) {
			t.Errorf("Usage output should contain env var %q\nGot:\n%s", env, output)
		}
	}

	// Verify type names.
	if !strings.Contains(output, "string") {
		t.Errorf("Usage output should contain type 'string'\nGot:\n%s", output)
	}
	if !strings.Contains(output, "int") {
		t.Errorf("Usage output should contain type 'int'\nGot:\n%s", output)
	}
	if !strings.Contains(output, "bool") {
		t.Errorf("Usage output should contain type 'bool'\nGot:\n%s", output)
	}

	// Verify default values.
	for _, def := range []string{"localhost", "5432", "info", "false"} {
		if !strings.Contains(output, def) {
			t.Errorf("Usage output should contain default %q\nGot:\n%s", def, output)
		}
	}

	// Verify descriptions.
	for _, desc := range []string{"database host", "database port", "logging level", "enable debug mode"} {
		if !strings.Contains(output, desc) {
			t.Errorf("Usage output should contain description %q\nGot:\n%s", desc, output)
		}
	}

	// Verify section header for nested struct.
	if !strings.Contains(output, "DB:") {
		t.Errorf("Usage output should contain section header 'DB:'\nGot:\n%s", output)
	}
}

// --- Combined end-to-end: realistic config scenario ---

func TestAcceptance_RealisticConfig_EndToEnd(t *testing.T) {
	type appConfig struct {
		Server struct {
			Host string `default:"0.0.0.0" description:"server bind address"  validate:"required"`
			Port int    `default:"8080"    description:"server port"           validate:"min=1,max=65535"`
		}
		DB struct {
			Host     string `default:"localhost" description:"database host"     validate:"required"`
			Port     int    `default:"5432"      description:"database port"     validate:"min=1,max=65535"`
			Name     string `default:"myapp"     description:"database name"     validate:"required"`
			Password string `default:""          description:"database password"`
		}
		LogLevel string   `default:"info"  description:"log level" validate:"oneof=debug info warn error"`
		Debug    bool     `default:"false" description:"debug mode"`
		Tags     []string `default:""`
	}

	// Write a YAML config file.
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "app.yaml")
	yamlContent := `
server:
  host: 127.0.0.1
  port: 9090
db:
  host: db.internal
  port: 3306
  name: production
  password: secret
log_level: warn
tags:
  - web
  - api
`
	if err := os.WriteFile(cfgFile, []byte(yamlContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// Env overrides db password and log level.
	t.Setenv("MYAPP_DB_PASSWORD", "env-secret")
	t.Setenv("MYAPP_LOG_LEVEL", "error")

	// Flag overrides server port.
	var cfg appConfig
	err := Load(&cfg,
		WithFile(cfgFile),
		WithEnvPrefix("MYAPP"),
		WithFlags([]string{"--server-port", "4443", "--debug", "true"}),
	)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	// Verify final values.
	checks := []struct {
		name string
		got  any
		want any
	}{
		{"Server.Host", cfg.Server.Host, "127.0.0.1"},
		{"Server.Port", cfg.Server.Port, 4443},
		{"DB.Host", cfg.DB.Host, "db.internal"},
		{"DB.Port", cfg.DB.Port, 3306},
		{"DB.Name", cfg.DB.Name, "production"},
		{"DB.Password", cfg.DB.Password, "env-secret"},
		{"LogLevel", cfg.LogLevel, "error"},
		{"Debug", cfg.Debug, true},
	}

	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s = %v, want %v", c.name, c.got, c.want)
		}
	}

	// Tags from file.
	if len(cfg.Tags) != 2 || cfg.Tags[0] != "web" || cfg.Tags[1] != "api" {
		t.Errorf("Tags = %v, want [web api]", cfg.Tags)
	}

	// Usage should work with same config.
	usage := Usage(&cfg, WithEnvPrefix("MYAPP"))
	if usage == "" {
		t.Error("Usage() returned empty string")
	}
	if !strings.Contains(usage, "Server:") || !strings.Contains(usage, "DB:") {
		t.Errorf("Usage should have section headers, got:\n%s", usage)
	}
}

// --- Edge cases ---

func TestAcceptance_WithFileContent_AllFormats(t *testing.T) {
	type cfg struct {
		Name  string `default:"default"`
		Count int    `default:"0"`
	}

	tests := []struct {
		name   string
		data   []byte
		format Format
	}{
		{"json", []byte(`{"name":"content","count":42}`), JSON},
		{"yaml", []byte("name: content\ncount: 42\n"), YAML},
		{"toml", []byte("name = \"content\"\ncount = 42\n"), TOML},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var c cfg
			err := Load(&c, WithFileContent(tt.data, tt.format))
			if err != nil {
				t.Fatalf("Load error: %v", err)
			}
			if c.Name != "content" {
				t.Errorf("Name = %q, want %q", c.Name, "content")
			}
			if c.Count != 42 {
				t.Errorf("Count = %d, want %d", c.Count, 42)
			}
		})
	}
}

func TestAcceptance_EmptyFile_DefaultsApply(t *testing.T) {
	type cfg struct {
		Host string `default:"localhost"`
		Port int    `default:"8080"`
	}

	var c cfg
	err := Load(&c, WithFile("testdata/empty.json"))
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if c.Host != "localhost" {
		t.Errorf("Host = %q, want %q", c.Host, "localhost")
	}
	if c.Port != 8080 {
		t.Errorf("Port = %d, want %d", c.Port, 8080)
	}
}

func TestAcceptance_ErrorTypes_AreCorrect(t *testing.T) {
	type cfg struct {
		Host string `validate:"required"`
	}

	// ErrInvalidTarget
	t.Run("ErrInvalidTarget", func(t *testing.T) {
		err := Load(nil)
		if !errors.Is(err, ErrInvalidTarget) {
			t.Errorf("expected ErrInvalidTarget, got %v", err)
		}
	})

	// ErrFileNotFound
	t.Run("ErrFileNotFound", func(t *testing.T) {
		var c cfg
		err := Load(&c, WithFile("/no/such/file.yaml"))
		if !errors.Is(err, ErrFileNotFound) {
			t.Errorf("expected ErrFileNotFound, got %v", err)
		}
	})

	// ErrParse
	t.Run("ErrParse", func(t *testing.T) {
		type intCfg struct {
			Port int `default:"0"`
		}
		t.Setenv("PORT", "not-a-number")
		var c intCfg
		err := Load(&c)
		if !errors.Is(err, ErrParse) {
			t.Errorf("expected ErrParse, got %v", err)
		}
	})

	// ErrValidation
	t.Run("ErrValidation", func(t *testing.T) {
		var c cfg
		err := Load(&c)
		if !errors.Is(err, ErrValidation) {
			t.Errorf("expected ErrValidation, got %v", err)
		}
	})
}
