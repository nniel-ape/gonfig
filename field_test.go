package gonfig

import (
	"reflect"
	"testing"
	"time"
)

func TestExtractFields_FlatStruct(t *testing.T) {
	type Config struct {
		Host    string
		Port    int
		Debug   bool
		Rate    float64
		Timeout time.Duration
	}

	var cfg Config
	fields := extractFields(reflect.ValueOf(cfg), "", nil)

	if len(fields) != 5 {
		t.Fatalf("expected 5 fields, got %d", len(fields))
	}

	tests := []struct {
		idx       int
		name      string
		path      string
		envName   string
		flagName  string
		configKey string
	}{
		{0, "Host", "Host", "HOST", "host", "host"},
		{1, "Port", "Port", "PORT", "port", "port"},
		{2, "Debug", "Debug", "DEBUG", "debug", "debug"},
		{3, "Rate", "Rate", "RATE", "rate", "rate"},
		{4, "Timeout", "Timeout", "TIMEOUT", "timeout", "timeout"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := fields[tt.idx]
			if f.Name != tt.name {
				t.Errorf("Name = %q, want %q", f.Name, tt.name)
			}
			if f.Path != tt.path {
				t.Errorf("Path = %q, want %q", f.Path, tt.path)
			}
			if f.EnvName != tt.envName {
				t.Errorf("EnvName = %q, want %q", f.EnvName, tt.envName)
			}
			if f.FlagName != tt.flagName {
				t.Errorf("FlagName = %q, want %q", f.FlagName, tt.flagName)
			}
			if f.ConfigKey != tt.configKey {
				t.Errorf("ConfigKey = %q, want %q", f.ConfigKey, tt.configKey)
			}
		})
	}
}

func TestExtractFields_NestedStruct(t *testing.T) {
	type DB struct {
		Host string
		Port int
	}
	type Config struct {
		DB       DB
		LogLevel string
	}

	var cfg Config
	fields := extractFields(reflect.ValueOf(cfg), "", nil)

	if len(fields) != 3 {
		t.Fatalf("expected 3 fields (2 nested + 1 top), got %d", len(fields))
	}

	tests := []struct {
		idx       int
		name      string
		path      string
		envName   string
		flagName  string
		configKey string
	}{
		{0, "Host", "DB.Host", "DB_HOST", "db-host", "db.host"},
		{1, "Port", "DB.Port", "DB_PORT", "db-port", "db.port"},
		{2, "LogLevel", "LogLevel", "LOG_LEVEL", "log-level", "log_level"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			f := fields[tt.idx]
			if f.Name != tt.name {
				t.Errorf("Name = %q, want %q", f.Name, tt.name)
			}
			if f.Path != tt.path {
				t.Errorf("Path = %q, want %q", f.Path, tt.path)
			}
			if f.EnvName != tt.envName {
				t.Errorf("EnvName = %q, want %q", f.EnvName, tt.envName)
			}
			if f.FlagName != tt.flagName {
				t.Errorf("FlagName = %q, want %q", f.FlagName, tt.flagName)
			}
			if f.ConfigKey != tt.configKey {
				t.Errorf("ConfigKey = %q, want %q", f.ConfigKey, tt.configKey)
			}
		})
	}
}

func TestExtractFields_ExplicitTagOverrides(t *testing.T) {
	type Config struct {
		Host string `env:"CUSTOM_HOST" flag:"custom-host" gonfig:"custom_host"`
		Port int    `env:"MY_PORT"`
		Name string `flag:"app-name"`
		Key  string `gonfig:"api_key"`
	}

	var cfg Config
	fields := extractFields(reflect.ValueOf(cfg), "", nil)

	if len(fields) != 4 {
		t.Fatalf("expected 4 fields, got %d", len(fields))
	}

	tests := []struct {
		idx       int
		envName   string
		flagName  string
		configKey string
	}{
		{0, "CUSTOM_HOST", "custom-host", "custom_host"},
		{1, "MY_PORT", "port", "port"},        // only env overridden
		{2, "NAME", "app-name", "name"},        // only flag overridden
		{3, "KEY", "key", "api_key"},            // only gonfig overridden
	}

	for _, tt := range tests {
		f := fields[tt.idx]
		t.Run(f.Name, func(t *testing.T) {
			if f.EnvName != tt.envName {
				t.Errorf("EnvName = %q, want %q", f.EnvName, tt.envName)
			}
			if f.FlagName != tt.flagName {
				t.Errorf("FlagName = %q, want %q", f.FlagName, tt.flagName)
			}
			if f.ConfigKey != tt.configKey {
				t.Errorf("ConfigKey = %q, want %q", f.ConfigKey, tt.configKey)
			}
		})
	}
}

func TestExtractFields_Tags(t *testing.T) {
	type Config struct {
		Host string `default:"localhost" description:"database host" validate:"required"`
		Port int    `default:"5432" description:"database port" validate:"min=1,max=65535"`
		Name string // no tags
	}

	var cfg Config
	fields := extractFields(reflect.ValueOf(cfg), "", nil)

	// Host
	if !fields[0].HasDefault || fields[0].DefaultVal != "localhost" {
		t.Errorf("Host default: HasDefault=%v, DefaultVal=%q", fields[0].HasDefault, fields[0].DefaultVal)
	}
	if fields[0].Description != "database host" {
		t.Errorf("Host description = %q", fields[0].Description)
	}
	if fields[0].ValidateRules != "required" {
		t.Errorf("Host validate = %q", fields[0].ValidateRules)
	}

	// Port
	if !fields[1].HasDefault || fields[1].DefaultVal != "5432" {
		t.Errorf("Port default: HasDefault=%v, DefaultVal=%q", fields[1].HasDefault, fields[1].DefaultVal)
	}

	// Name — no default tag
	if fields[2].HasDefault {
		t.Error("Name should not have default")
	}
	if fields[2].Description != "" {
		t.Errorf("Name description should be empty, got %q", fields[2].Description)
	}
}

func TestExtractFields_UnexportedFieldsSkipped(t *testing.T) {
	type Config struct {
		Host     string
		internal string //nolint:unused
	}

	var cfg Config
	_ = cfg.internal // suppress unused field warning; field exists to test unexported-field skipping
	fields := extractFields(reflect.ValueOf(cfg), "", nil)

	if len(fields) != 1 {
		t.Fatalf("expected 1 field (unexported skipped), got %d", len(fields))
	}
	if fields[0].Name != "Host" {
		t.Errorf("expected Host, got %q", fields[0].Name)
	}
}

func TestExtractFields_IndexPath(t *testing.T) {
	type Inner struct {
		Value string
	}
	type Config struct {
		Name  string
		Inner Inner
	}

	var cfg Config
	fields := extractFields(reflect.ValueOf(cfg), "", nil)

	// Name is at index [0]
	if !reflect.DeepEqual(fields[0].Index, []int{0}) {
		t.Errorf("Name index = %v, want [0]", fields[0].Index)
	}
	// Inner.Value is at index [1, 0]
	if !reflect.DeepEqual(fields[1].Index, []int{1, 0}) {
		t.Errorf("Inner.Value index = %v, want [1, 0]", fields[1].Index)
	}
}

func TestExtractFields_ShortFlagTag(t *testing.T) {
	type Config struct {
		Port  int    `short:"p" description:"server port"`
		Debug bool   `short:"d" description:"enable debug"`
		Host  string `description:"server host"` // no short flag
	}

	var cfg Config
	fields := extractFields(reflect.ValueOf(cfg), "", nil)

	if len(fields) != 3 {
		t.Fatalf("expected 3 fields, got %d", len(fields))
	}

	if fields[0].ShortFlag != "p" {
		t.Errorf("Port ShortFlag = %q, want %q", fields[0].ShortFlag, "p")
	}
	if fields[1].ShortFlag != "d" {
		t.Errorf("Debug ShortFlag = %q, want %q", fields[1].ShortFlag, "d")
	}
	if fields[2].ShortFlag != "" {
		t.Errorf("Host ShortFlag = %q, want empty", fields[2].ShortFlag)
	}
}

func TestToEnvName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Host", "HOST"},
		{"DB.Host", "DB_HOST"},
		{"LogLevel", "LOG_LEVEL"},
		{"DB.MaxConn", "DB_MAX_CONN"},
		{"HTTPSPort", "HTTPS_PORT"},
		{"DBHost", "DB_HOST"},
		{"SimpleURL", "SIMPLE_URL"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := toEnvName(tt.input)
			if got != tt.want {
				t.Errorf("toEnvName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestToFlagName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Host", "host"},
		{"DB.Host", "db-host"},
		{"LogLevel", "log-level"},
		{"DB.MaxConn", "db-max-conn"},
		{"HTTPSPort", "https-port"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := toFlagName(tt.input)
			if got != tt.want {
				t.Errorf("toFlagName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestToConfigKey(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Host", "host"},
		{"DB.Host", "db.host"},
		{"LogLevel", "log_level"},
		{"DB.MaxConn", "db.max_conn"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := toConfigKey(tt.input)
			if got != tt.want {
				t.Errorf("toConfigKey(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestCamelToSnake(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Host", "Host"},
		{"LogLevel", "Log_Level"},
		{"MaxConn", "Max_Conn"},
		{"DBHost", "DB_Host"},
		{"HTTPSPort", "HTTPS_Port"},
		{"SimpleURL", "Simple_URL"},
		{"ID", "ID"},
		{"", ""},
		{"a", "a"},
		{"A", "A"},
		{"Ab", "Ab"},
		{"AB", "AB"},
		{"ABc", "A_Bc"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := camelToSnake(tt.input)
			if got != tt.want {
				t.Errorf("camelToSnake(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
