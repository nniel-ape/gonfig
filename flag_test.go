package gonfig

import (
	"errors"
	"flag"
	"reflect"
	"testing"
	"time"
)

func TestApplyFlags_BasicTypes(t *testing.T) {
	type Config struct {
		Host    string
		Port    int
		Debug   bool
		Rate    float64
		Timeout time.Duration
	}

	tests := []struct {
		name string
		args []string
		want Config
	}{
		{
			name: "string flag",
			args: []string{"--host", "flaghost"},
			want: Config{Host: "flaghost"},
		},
		{
			name: "int flag",
			args: []string{"--port", "9090"},
			want: Config{Port: 9090},
		},
		{
			name: "bool flag",
			args: []string{"--debug", "true"},
			want: Config{Debug: true},
		},
		{
			name: "float flag",
			args: []string{"--rate", "2.5"},
			want: Config{Rate: 2.5},
		},
		{
			name: "duration flag",
			args: []string{"--timeout", "30s"},
			want: Config{Timeout: 30 * time.Second},
		},
		{
			name: "multiple flags",
			args: []string{"--host", "multi", "--port", "7070", "--debug", "true"},
			want: Config{Host: "multi", Port: 7070, Debug: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cfg Config
			v := reflect.ValueOf(&cfg).Elem()
			fields := extractFields(v, "", nil)

			if err := applyFlags(&cfg, fields, tt.args); err != nil {
				t.Fatalf("applyFlags() error = %v", err)
			}

			if cfg != tt.want {
				t.Errorf("applyFlags() got = %+v, want = %+v", cfg, tt.want)
			}
		})
	}
}

func TestApplyFlags_OnlyExplicitOverride(t *testing.T) {
	type Config struct {
		Host string
		Port int
		Name string
	}

	// Pre-set values simulating earlier sources (file/env).
	cfg := Config{
		Host: "from-env",
		Port: 5432,
		Name: "from-file",
	}

	// Only --port is passed; Host and Name should be preserved.
	args := []string{"--port", "9999"}

	v := reflect.ValueOf(&cfg).Elem()
	fields := extractFields(v, "", nil)

	if err := applyFlags(&cfg, fields, args); err != nil {
		t.Fatalf("applyFlags() error = %v", err)
	}

	if cfg.Host != "from-env" {
		t.Errorf("Host = %q, want %q (should not be overridden)", cfg.Host, "from-env")
	}
	if cfg.Port != 9999 {
		t.Errorf("Port = %d, want %d", cfg.Port, 9999)
	}
	if cfg.Name != "from-file" {
		t.Errorf("Name = %q, want %q (should not be overridden)", cfg.Name, "from-file")
	}
}

func TestApplyFlags_ZeroValueExplicitlySet(t *testing.T) {
	type Config struct {
		Port  int
		Debug bool
		Host  string
	}

	// Pre-set non-zero values.
	cfg := Config{
		Port:  8080,
		Debug: true,
		Host:  "original",
	}

	// Explicitly set flags to zero values — these must override.
	args := []string{"--port", "0", "--debug", "false", "--host", ""}

	v := reflect.ValueOf(&cfg).Elem()
	fields := extractFields(v, "", nil)

	if err := applyFlags(&cfg, fields, args); err != nil {
		t.Fatalf("applyFlags() error = %v", err)
	}

	if cfg.Port != 0 {
		t.Errorf("Port = %d, want 0 (explicitly set to zero)", cfg.Port)
	}
	if cfg.Debug != false {
		t.Errorf("Debug = %v, want false (explicitly set to false)", cfg.Debug)
	}
	if cfg.Host != "" {
		t.Errorf("Host = %q, want empty string (explicitly set to empty)", cfg.Host)
	}
}

func TestApplyFlags_NestedStruct(t *testing.T) {
	type Config struct {
		DB struct {
			Host string
			Port int
		}
		LogLevel string
	}

	args := []string{"--db-host", "dbhost", "--db-port", "3306", "--log-level", "debug"}

	var cfg Config
	v := reflect.ValueOf(&cfg).Elem()
	fields := extractFields(v, "", nil)

	if err := applyFlags(&cfg, fields, args); err != nil {
		t.Fatalf("applyFlags() error = %v", err)
	}

	if cfg.DB.Host != "dbhost" {
		t.Errorf("DB.Host = %q, want %q", cfg.DB.Host, "dbhost")
	}
	if cfg.DB.Port != 3306 {
		t.Errorf("DB.Port = %d, want %d", cfg.DB.Port, 3306)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, "debug")
	}
}

func TestApplyFlags_ExplicitFlagTag(t *testing.T) {
	type Config struct {
		Host string `flag:"server-host"`
		Port int    `flag:"server-port"`
	}

	args := []string{"--server-host", "tagged", "--server-port", "4444"}

	var cfg Config
	v := reflect.ValueOf(&cfg).Elem()
	fields := extractFields(v, "", nil)

	if err := applyFlags(&cfg, fields, args); err != nil {
		t.Fatalf("applyFlags() error = %v", err)
	}

	if cfg.Host != "tagged" {
		t.Errorf("Host = %q, want %q", cfg.Host, "tagged")
	}
	if cfg.Port != 4444 {
		t.Errorf("Port = %d, want %d", cfg.Port, 4444)
	}
}

func TestApplyFlags_UnknownFlag(t *testing.T) {
	type Config struct {
		Host string
	}

	args := []string{"--unknown-flag", "value"}

	var cfg Config
	v := reflect.ValueOf(&cfg).Elem()
	fields := extractFields(v, "", nil)

	err := applyFlags(&cfg, fields, args)
	if err == nil {
		t.Fatal("applyFlags() expected error for unknown flag, got nil")
	}
}

func TestApplyFlags_Help(t *testing.T) {
	type Config struct {
		Host string `description:"server hostname"`
		Port int    `description:"server port"`
	}

	args := []string{"--help"}

	var cfg Config
	v := reflect.ValueOf(&cfg).Elem()
	fields := extractFields(v, "", nil)

	err := applyFlags(&cfg, fields, args)
	if err == nil {
		t.Fatal("applyFlags() expected error for --help, got nil")
	}
	if !errors.Is(err, flag.ErrHelp) {
		t.Errorf("applyFlags() error = %v, want wrapped flag.ErrHelp", err)
	}
}

func TestApplyFlags_NoArgs(t *testing.T) {
	type Config struct {
		Host string
		Port int
	}

	cfg := Config{Host: "preserved", Port: 1234}

	v := reflect.ValueOf(&cfg).Elem()
	fields := extractFields(v, "", nil)

	if err := applyFlags(&cfg, fields, nil); err != nil {
		t.Fatalf("applyFlags() error = %v", err)
	}

	if cfg.Host != "preserved" {
		t.Errorf("Host = %q, want %q", cfg.Host, "preserved")
	}
	if cfg.Port != 1234 {
		t.Errorf("Port = %d, want %d", cfg.Port, 1234)
	}
}

func TestApplyFlags_InvalidValue(t *testing.T) {
	type Config struct {
		Port int
	}

	args := []string{"--port", "not-a-number"}

	var cfg Config
	v := reflect.ValueOf(&cfg).Elem()
	fields := extractFields(v, "", nil)

	err := applyFlags(&cfg, fields, args)
	if err == nil {
		t.Fatal("applyFlags() expected error for invalid int value, got nil")
	}
}

func TestApplyFlags_EqualsSyntax(t *testing.T) {
	type Config struct {
		Host string
		Port int
	}

	args := []string{"--host=eqhost", "--port=5555"}

	var cfg Config
	v := reflect.ValueOf(&cfg).Elem()
	fields := extractFields(v, "", nil)

	if err := applyFlags(&cfg, fields, args); err != nil {
		t.Fatalf("applyFlags() error = %v", err)
	}

	if cfg.Host != "eqhost" {
		t.Errorf("Host = %q, want %q", cfg.Host, "eqhost")
	}
	if cfg.Port != 5555 {
		t.Errorf("Port = %d, want %d", cfg.Port, 5555)
	}
}
