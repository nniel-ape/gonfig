package gonfig

import (
	"errors"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// testConfig is a nested config struct used across multiple tests.
type testConfig struct {
	DB struct {
		Host string `default:"localhost" description:"database host"`
		Port int    `default:"5432"      description:"database port"`
	}
	LogLevel string `default:"info"  description:"logging level"`
	Debug    bool   `default:"false" description:"enable debug mode"`
}

func TestLoad_AllSourcesCombined_PriorityOrder(t *testing.T) {
	// Create a temp YAML config file that sets db.host and log_level.
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(cfgFile, []byte("db:\n  host: filehost\nlog_level: warn\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Env overrides file for LogLevel.
	t.Setenv("APP_LOG_LEVEL", "error")

	// Flag overrides everything for db-host.
	var cfg testConfig
	err := Load(&cfg,
		WithFile(cfgFile),
		WithEnvPrefix("APP"),
		WithFlags([]string{"--db-host", "flaghost"}),
	)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	// db.host: default=localhost, file=filehost, flag=flaghost → flaghost wins
	if cfg.DB.Host != "flaghost" {
		t.Errorf("DB.Host = %q, want %q (flag should override file)", cfg.DB.Host, "flaghost")
	}

	// db.port: default=5432, not in file/env/flag → default wins
	if cfg.DB.Port != 5432 {
		t.Errorf("DB.Port = %d, want %d (default)", cfg.DB.Port, 5432)
	}

	// log_level: default=info, file=warn, env=error → env wins
	if cfg.LogLevel != "error" {
		t.Errorf("LogLevel = %q, want %q (env should override file)", cfg.LogLevel, "error")
	}

	// debug: default=false, not in file/env/flag → default wins
	if cfg.Debug != false {
		t.Errorf("Debug = %v, want false (default)", cfg.Debug)
	}
}

func TestLoad_OnlyDefaults(t *testing.T) {
	var cfg testConfig
	err := Load(&cfg)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.DB.Host != "localhost" {
		t.Errorf("DB.Host = %q, want %q", cfg.DB.Host, "localhost")
	}
	if cfg.DB.Port != 5432 {
		t.Errorf("DB.Port = %d, want %d", cfg.DB.Port, 5432)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, "info")
	}
	if cfg.Debug != false {
		t.Errorf("Debug = %v, want false", cfg.Debug)
	}
}

func TestLoad_FileAndEnvOverride(t *testing.T) {
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config.json")
	if err := os.WriteFile(cfgFile, []byte(`{"db":{"host":"jsonhost","port":3306},"log_level":"debug"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	// Env overrides file for db.host.
	t.Setenv("DB_HOST", "envhost")

	var cfg testConfig
	err := Load(&cfg,
		WithFile(cfgFile),
	)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	// db.host: file=jsonhost, env=envhost → env wins (no prefix, so DB_HOST matches)
	if cfg.DB.Host != "envhost" {
		t.Errorf("DB.Host = %q, want %q (env should override file)", cfg.DB.Host, "envhost")
	}

	// db.port: file=3306 → file value
	if cfg.DB.Port != 3306 {
		t.Errorf("DB.Port = %d, want %d (from file)", cfg.DB.Port, 3306)
	}

	// log_level: file=debug → file value (no env set for it)
	if cfg.LogLevel != "debug" {
		t.Errorf("LogLevel = %q, want %q (from file)", cfg.LogLevel, "debug")
	}
}

func TestLoad_FlagOverridesEverything(t *testing.T) {
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(cfgFile, []byte("log_level = \"warn\"\n\n[db]\nhost = \"tomlhost\"\nport = 9999\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("MYAPP_LOG_LEVEL", "error")

	var cfg testConfig
	err := Load(&cfg,
		WithFile(cfgFile),
		WithEnvPrefix("MYAPP"),
		WithFlags([]string{"--log-level", "trace", "--db-port", "1234"}),
	)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	// log_level: default=info, file=warn, env=error, flag=trace → flag wins
	if cfg.LogLevel != "trace" {
		t.Errorf("LogLevel = %q, want %q (flag should override all)", cfg.LogLevel, "trace")
	}

	// db.port: default=5432, file=9999, flag=1234 → flag wins
	if cfg.DB.Port != 1234 {
		t.Errorf("DB.Port = %d, want %d (flag should override all)", cfg.DB.Port, 1234)
	}

	// db.host: file=tomlhost, no flag → file value
	if cfg.DB.Host != "tomlhost" {
		t.Errorf("DB.Host = %q, want %q (from file, no flag override)", cfg.DB.Host, "tomlhost")
	}
}

func TestLoad_NilTarget(t *testing.T) {
	err := Load(nil)
	if !errors.Is(err, ErrInvalidTarget) {
		t.Errorf("Load(nil) = %v, want ErrInvalidTarget", err)
	}
}

func TestLoad_NonPointerTarget(t *testing.T) {
	var cfg testConfig
	err := Load(cfg) // not a pointer
	if !errors.Is(err, ErrInvalidTarget) {
		t.Errorf("Load(non-pointer) = %v, want ErrInvalidTarget", err)
	}
}

func TestLoad_NonStructTarget(t *testing.T) {
	s := "not a struct"
	err := Load(&s)
	if !errors.Is(err, ErrInvalidTarget) {
		t.Errorf("Load(&string) = %v, want ErrInvalidTarget", err)
	}
}

func TestLoad_PointerToInt(t *testing.T) {
	n := 42
	err := Load(&n)
	if !errors.Is(err, ErrInvalidTarget) {
		t.Errorf("Load(&int) = %v, want ErrInvalidTarget", err)
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	var cfg testConfig
	err := Load(&cfg, WithFile("/nonexistent/path/config.yaml"))
	if !errors.Is(err, ErrFileNotFound) {
		t.Errorf("Load(missing file) = %v, want ErrFileNotFound", err)
	}
}

func TestLoad_WithFileContent_JSON(t *testing.T) {
	data := []byte(`{"db":{"host":"contenthost","port":7777},"log_level":"debug"}`)
	var cfg testConfig
	err := Load(&cfg, WithFileContent(data, JSON))
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.DB.Host != "contenthost" {
		t.Errorf("DB.Host = %q, want %q", cfg.DB.Host, "contenthost")
	}
	if cfg.DB.Port != 7777 {
		t.Errorf("DB.Port = %d, want %d", cfg.DB.Port, 7777)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, "debug")
	}
}

func TestLoad_WithFileContent_YAML(t *testing.T) {
	data := []byte("db:\n  host: yamlhost\nlog_level: warn\n")
	var cfg testConfig
	err := Load(&cfg, WithFileContent(data, YAML))
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.DB.Host != "yamlhost" {
		t.Errorf("DB.Host = %q, want %q", cfg.DB.Host, "yamlhost")
	}
	if cfg.LogLevel != "warn" {
		t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, "warn")
	}
}

func TestLoad_WithFileContent_TOML(t *testing.T) {
	data := []byte("log_level = \"tomlval\"\n\n[db]\nhost = \"tomlhost\"\nport = 2222\n")
	var cfg testConfig
	err := Load(&cfg, WithFileContent(data, TOML))
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.DB.Host != "tomlhost" {
		t.Errorf("DB.Host = %q, want %q", cfg.DB.Host, "tomlhost")
	}
	if cfg.DB.Port != 2222 {
		t.Errorf("DB.Port = %d, want %d", cfg.DB.Port, 2222)
	}
}

func TestLoad_WithFileContent_InvalidJSON(t *testing.T) {
	data := []byte(`{invalid json}`)
	var cfg testConfig
	err := Load(&cfg, WithFileContent(data, JSON))
	if !errors.Is(err, ErrParse) {
		t.Errorf("Load(invalid content) = %v, want ErrParse", err)
	}
}

func TestLoad_EmptyStruct(t *testing.T) {
	type Empty struct{}
	var cfg Empty
	err := Load(&cfg)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
}

func TestLoad_WithFlags_EmptyArgs(t *testing.T) {
	var cfg testConfig
	err := Load(&cfg, WithFlags([]string{}))
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	// Should have defaults only.
	if cfg.DB.Host != "localhost" {
		t.Errorf("DB.Host = %q, want %q", cfg.DB.Host, "localhost")
	}
}

func TestLoad_EnvWithPrefix(t *testing.T) {
	t.Setenv("MYAPP_DB_HOST", "prefixhost")
	t.Setenv("MYAPP_DB_PORT", "9999")

	var cfg testConfig
	err := Load(&cfg, WithEnvPrefix("MYAPP"))
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.DB.Host != "prefixhost" {
		t.Errorf("DB.Host = %q, want %q", cfg.DB.Host, "prefixhost")
	}
	if cfg.DB.Port != 9999 {
		t.Errorf("DB.Port = %d, want %d", cfg.DB.Port, 9999)
	}
}

func TestLoad_MultipleFiles_LaterOverridesEarlier(t *testing.T) {
	dir := t.TempDir()

	file1 := filepath.Join(dir, "base.json")
	if err := os.WriteFile(file1, []byte(`{"db":{"host":"base","port":1111},"log_level":"info"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	file2 := filepath.Join(dir, "override.json")
	if err := os.WriteFile(file2, []byte(`{"db":{"host":"override"}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	var cfg testConfig
	err := Load(&cfg, WithFile(file1), WithFile(file2))
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	// db.host: file1=base, file2=override → override
	if cfg.DB.Host != "override" {
		t.Errorf("DB.Host = %q, want %q", cfg.DB.Host, "override")
	}

	// db.port: file1=1111, file2 doesn't set it → 1111
	if cfg.DB.Port != 1111 {
		t.Errorf("DB.Port = %d, want %d", cfg.DB.Port, 1111)
	}
}

func TestLoad_InterleavedFileContentAndFile_PreservesOrder(t *testing.T) {
	dir := t.TempDir()

	// File sets db.host to "fromfile"
	cfgFile := filepath.Join(dir, "config.json")
	if err := os.WriteFile(cfgFile, []byte(`{"db":{"host":"fromfile"}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	// Inline content is specified first, file second — file should win (later overrides earlier).
	contentData := []byte(`{"db":{"host":"frominline"}}`)
	var cfg testConfig
	err := Load(&cfg, WithFileContent(contentData, JSON), WithFile(cfgFile))
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.DB.Host != "fromfile" {
		t.Errorf("DB.Host = %q, want %q (WithFile specified after WithFileContent should win)", cfg.DB.Host, "fromfile")
	}
}

func TestLoad_InterleavedFileAndFileContent_PreservesOrder(t *testing.T) {
	dir := t.TempDir()

	// File sets db.host to "fromfile"
	cfgFile := filepath.Join(dir, "config.json")
	if err := os.WriteFile(cfgFile, []byte(`{"db":{"host":"fromfile"}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	// File is specified first, inline content second — inline should win (later overrides earlier).
	contentData := []byte(`{"db":{"host":"frominline"}}`)
	var cfg testConfig
	err := Load(&cfg, WithFile(cfgFile), WithFileContent(contentData, JSON))
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.DB.Host != "frominline" {
		t.Errorf("DB.Host = %q, want %q (WithFileContent specified after WithFile should win)", cfg.DB.Host, "frominline")
	}
}

func TestLoad_InvalidEnvValue(t *testing.T) {
	t.Setenv("DB_PORT", "not-a-number")

	var cfg testConfig
	err := Load(&cfg)
	if !errors.Is(err, ErrParse) {
		t.Errorf("Load(invalid env) = %v, want ErrParse", err)
	}
}

func TestLoad_InvalidFlagValue(t *testing.T) {
	var cfg testConfig
	err := Load(&cfg, WithFlags([]string{"--db-port", "not-a-number"}))
	if !errors.Is(err, ErrParse) {
		t.Errorf("Load(invalid flag) = %v, want ErrParse", err)
	}
}

func TestLoad_WithTestdataFiles(t *testing.T) {
	// Use the existing testdata fixtures.
	tests := []struct {
		name     string
		file     string
		wantHost string
		wantPort int
	}{
		{"json", "testdata/nested.json", "dbhost", 5432},
		{"yaml", "testdata/nested.yaml", "dbhost", 5432},
		{"toml", "testdata/nested.toml", "dbhost", 5432},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cfg testConfig
			err := Load(&cfg, WithFile(tt.file))
			if err != nil {
				t.Fatalf("Load() error: %v", err)
			}

			if cfg.DB.Host != tt.wantHost {
				t.Errorf("DB.Host = %q, want %q", cfg.DB.Host, tt.wantHost)
			}
			if cfg.DB.Port != tt.wantPort {
				t.Errorf("DB.Port = %d, want %d", cfg.DB.Port, tt.wantPort)
			}
		})
	}
}

func TestLoad_UnsupportedFileFormat(t *testing.T) {
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config.ini")
	if err := os.WriteFile(cfgFile, []byte("[section]\nkey=value\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var cfg testConfig
	err := Load(&cfg, WithFile(cfgFile))
	if err == nil {
		t.Error("Load(unsupported format) should return error")
	}
	if !errors.Is(err, ErrParse) {
		t.Errorf("Load(unsupported format) should return ErrParse, got: %v", err)
	}
}

// swapAutoHelp overrides osExit and printFn for testing auto-help behavior.
// Returns a cleanup function (also registered via t.Cleanup).
func swapAutoHelp(t *testing.T) (exitCode *int, printed *string) {
	t.Helper()
	oldExit := osExit
	oldPrint := printFn

	code := -1
	var out string
	osExit = func(c int) { code = c }
	printFn = func(s string) { out = s }
	t.Cleanup(func() {
		osExit = oldExit
		printFn = oldPrint
	})
	return &code, &out
}

func TestLoad_AutoHelp_PrintsUsageAndExits(t *testing.T) {
	exitCode, printed := swapAutoHelp(t)

	type Config struct {
		Host string `default:"localhost" description:"server host"`
		Port int    `default:"8080"      description:"server port"`
	}

	var cfg Config
	err := Load(&cfg,
		WithFlags([]string{"--help"}),
		WithEnvPrefix("APP"),
	)
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}
	if *exitCode != 0 {
		t.Errorf("exit code = %d, want 0", *exitCode)
	}
	if !strings.Contains(*printed, "APP_HOST") {
		t.Errorf("usage should contain APP_HOST, got: %s", *printed)
	}
	if !strings.Contains(*printed, "--host") {
		t.Errorf("usage should contain --host, got: %s", *printed)
	}
}

func TestLoad_AutoHelp_ShortFlag(t *testing.T) {
	exitCode, printed := swapAutoHelp(t)

	type Config struct {
		Host string `default:"localhost" description:"host"`
	}

	var cfg Config
	err := Load(&cfg, WithFlags([]string{"-h"}))
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}
	if *exitCode != 0 {
		t.Errorf("exit code = %d, want 0", *exitCode)
	}
	if *printed == "" {
		t.Error("expected usage output, got empty string")
	}
}

func TestLoad_AutoHelpDisabled_ReturnsFlagErrHelp(t *testing.T) {
	type Config struct {
		Host string `default:"localhost"`
	}

	var cfg Config
	err := Load(&cfg,
		WithFlags([]string{"--help"}),
		WithAutoHelp(false),
	)
	if !errors.Is(err, flag.ErrHelp) {
		t.Errorf("Load() error = %v, want flag.ErrHelp", err)
	}
}

func TestLoad_WithoutValidation_SkipsValidation(t *testing.T) {
	type Config struct {
		Port int `default:"0" validate:"required,min=1"`
	}

	var cfg Config
	err := Load(&cfg, WithoutValidation())
	if err != nil {
		t.Fatalf("Load() should not return error when validation is skipped, got: %v", err)
	}
	if cfg.Port != 0 {
		t.Errorf("Port = %d, want 0 (default)", cfg.Port)
	}
}

func TestLoad_ValidationRunsByDefault(t *testing.T) {
	type Config struct {
		Port int `default:"0" validate:"required"`
	}

	var cfg Config
	err := Load(&cfg)
	if !errors.Is(err, ErrValidation) {
		t.Errorf("Load() error = %v, want ErrValidation", err)
	}
}
