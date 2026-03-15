package gonfig

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestDecodeJSON(t *testing.T) {
	input := `{"host": "localhost", "port": 8080, "debug": true}`
	data, err := decodeJSON(strings.NewReader(input))
	if err != nil {
		t.Fatalf("decodeJSON: unexpected error: %v", err)
	}
	if data["host"] != "localhost" {
		t.Errorf("host = %v, want localhost", data["host"])
	}
	if data["port"] != float64(8080) {
		t.Errorf("port = %v, want 8080", data["port"])
	}
	if data["debug"] != true {
		t.Errorf("debug = %v, want true", data["debug"])
	}
}

func TestDecodeJSON_Invalid(t *testing.T) {
	_, err := decodeJSON(strings.NewReader("not json"))
	if err == nil {
		t.Fatal("decodeJSON: expected error for invalid JSON")
	}
}

func TestLookupMap(t *testing.T) {
	data := map[string]any{
		"host": "localhost",
		"db": map[string]any{
			"host": "dbhost",
			"port": float64(5432),
		},
	}

	tests := []struct {
		key    string
		want   any
		wantOK bool
	}{
		{"host", "localhost", true},
		{"db.host", "dbhost", true},
		{"db.port", float64(5432), true},
		{"missing", nil, false},
		{"db.missing", nil, false},
		{"db.host.extra", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			got, ok := lookupMap(data, tt.key)
			if ok != tt.wantOK {
				t.Errorf("lookupMap(%q) ok = %v, want %v", tt.key, ok, tt.wantOK)
			}
			if ok && got != tt.want {
				t.Errorf("lookupMap(%q) = %v, want %v", tt.key, got, tt.want)
			}
		})
	}
}

func TestSetFieldFromAny(t *testing.T) {
	tests := []struct {
		name    string
		typ     reflect.Type
		val     any
		want    any
		wantErr bool
	}{
		{"string", reflect.TypeFor[string](), "hello", "hello", false},
		{"int from float64", reflect.TypeFor[int](), float64(42), int(42), false},
		{"int64 from float64", reflect.TypeFor[int64](), float64(99), int64(99), false},
		{"float64", reflect.TypeFor[float64](), float64(3.14), float64(3.14), false},
		{"bool true", reflect.TypeFor[bool](), true, true, false},
		{"bool false", reflect.TypeFor[bool](), false, false, false},
		{"duration", reflect.TypeFor[time.Duration](), "5s", 5 * time.Second, false},
		{"string type mismatch", reflect.TypeFor[string](), 42, nil, true},
		{"int type mismatch", reflect.TypeFor[int](), "not a number", nil, true},
		{"bool type mismatch", reflect.TypeFor[bool](), "not a bool", nil, true},
		{"duration type mismatch", reflect.TypeFor[time.Duration](), 42, nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field := reflect.New(tt.typ).Elem()
			err := setFieldFromAny(field, tt.val)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			got := field.Interface()
			if got != tt.want {
				t.Errorf("got %v (%T), want %v (%T)", got, got, tt.want, tt.want)
			}
		})
	}
}

func TestSetSliceFromAny(t *testing.T) {
	t.Run("string slice", func(t *testing.T) {
		field := reflect.New(reflect.TypeFor[[]string]()).Elem()
		err := setSliceFromAny(field, []any{"a", "b", "c"}, field.Type())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got := field.Interface().([]string)
		want := []string{"a", "b", "c"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("int slice", func(t *testing.T) {
		field := reflect.New(reflect.TypeFor[[]int]()).Elem()
		err := setSliceFromAny(field, []any{float64(1), float64(2), float64(3)}, field.Type())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got := field.Interface().([]int)
		want := []int{1, 2, 3}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("not an array", func(t *testing.T) {
		field := reflect.New(reflect.TypeFor[[]string]()).Elem()
		err := setSliceFromAny(field, "not an array", field.Type())
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestApplyMap_Flat(t *testing.T) {
	type Config struct {
		Host     string  `gonfig:"host"`
		Port     int     `gonfig:"port"`
		Debug    bool    `gonfig:"debug"`
		LogLevel string  `gonfig:"log_level"`
		Rate     float64 `gonfig:"rate"`
	}

	var cfg Config
	fields := extractFields(reflect.ValueOf(&cfg).Elem(), "", nil)

	data := map[string]any{
		"host":      "localhost",
		"port":      float64(8080),
		"debug":     true,
		"log_level": "debug",
		"rate":      float64(3.14),
	}

	if err := applyMap(&cfg, data, fields); err != nil {
		t.Fatalf("applyMap: unexpected error: %v", err)
	}

	if cfg.Host != "localhost" {
		t.Errorf("Host = %q, want %q", cfg.Host, "localhost")
	}
	if cfg.Port != 8080 {
		t.Errorf("Port = %d, want %d", cfg.Port, 8080)
	}
	if cfg.Debug != true {
		t.Errorf("Debug = %v, want true", cfg.Debug)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, "debug")
	}
	if cfg.Rate != 3.14 {
		t.Errorf("Rate = %f, want %f", cfg.Rate, 3.14)
	}
}

func TestApplyMap_Nested(t *testing.T) {
	type Config struct {
		DB struct {
			Host string
			Port int
		}
		LogLevel string
		Debug    bool
	}

	var cfg Config
	fields := extractFields(reflect.ValueOf(&cfg).Elem(), "", nil)

	data := map[string]any{
		"db": map[string]any{
			"host": "dbhost",
			"port": float64(5432),
		},
		"log_level": "warn",
		"debug":     false,
	}

	if err := applyMap(&cfg, data, fields); err != nil {
		t.Fatalf("applyMap: unexpected error: %v", err)
	}

	if cfg.DB.Host != "dbhost" {
		t.Errorf("DB.Host = %q, want %q", cfg.DB.Host, "dbhost")
	}
	if cfg.DB.Port != 5432 {
		t.Errorf("DB.Port = %d, want %d", cfg.DB.Port, 5432)
	}
	if cfg.LogLevel != "warn" {
		t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, "warn")
	}
}

func TestApplyMap_EmptyMap(t *testing.T) {
	type Config struct {
		Host string
		Port int
	}

	cfg := Config{Host: "original", Port: 1234}
	fields := extractFields(reflect.ValueOf(&cfg).Elem(), "", nil)

	if err := applyMap(&cfg, map[string]any{}, fields); err != nil {
		t.Fatalf("applyMap: unexpected error: %v", err)
	}

	// Fields should remain unchanged.
	if cfg.Host != "original" {
		t.Errorf("Host = %q, want %q", cfg.Host, "original")
	}
	if cfg.Port != 1234 {
		t.Errorf("Port = %d, want %d", cfg.Port, 1234)
	}
}

func TestLoadFile_JSON_Flat(t *testing.T) {
	type Config struct {
		Host     string  `gonfig:"host"`
		Port     int     `gonfig:"port"`
		Debug    bool    `gonfig:"debug"`
		LogLevel string  `gonfig:"log_level"`
		Rate     float64 `gonfig:"rate"`
	}

	var cfg Config
	fields := extractFields(reflect.ValueOf(&cfg).Elem(), "", nil)

	if err := loadFile(&cfg, "testdata/valid.json", fields); err != nil {
		t.Fatalf("loadFile: unexpected error: %v", err)
	}

	if cfg.Host != "localhost" {
		t.Errorf("Host = %q, want %q", cfg.Host, "localhost")
	}
	if cfg.Port != 8080 {
		t.Errorf("Port = %d, want %d", cfg.Port, 8080)
	}
	if cfg.Debug != true {
		t.Errorf("Debug = %v, want true", cfg.Debug)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, "debug")
	}
	if cfg.Rate != 3.14 {
		t.Errorf("Rate = %f, want %f", cfg.Rate, 3.14)
	}
}

func TestLoadFile_JSON_Nested(t *testing.T) {
	type Config struct {
		DB struct {
			Host string
			Port int
		}
		LogLevel string
		Debug    bool
	}

	var cfg Config
	fields := extractFields(reflect.ValueOf(&cfg).Elem(), "", nil)

	if err := loadFile(&cfg, "testdata/nested.json", fields); err != nil {
		t.Fatalf("loadFile: unexpected error: %v", err)
	}

	if cfg.DB.Host != "dbhost" {
		t.Errorf("DB.Host = %q, want %q", cfg.DB.Host, "dbhost")
	}
	if cfg.DB.Port != 5432 {
		t.Errorf("DB.Port = %d, want %d", cfg.DB.Port, 5432)
	}
	if cfg.LogLevel != "warn" {
		t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, "warn")
	}
	if cfg.Debug != false {
		t.Errorf("Debug = %v, want false", cfg.Debug)
	}
}

func TestLoadFile_JSON_Empty(t *testing.T) {
	type Config struct {
		Host string
	}

	cfg := Config{Host: "original"}
	fields := extractFields(reflect.ValueOf(&cfg).Elem(), "", nil)

	if err := loadFile(&cfg, "testdata/empty.json", fields); err != nil {
		t.Fatalf("loadFile: unexpected error: %v", err)
	}

	if cfg.Host != "original" {
		t.Errorf("Host = %q, want %q (should be unchanged)", cfg.Host, "original")
	}
}

func TestLoadFile_FileNotFound(t *testing.T) {
	type Config struct {
		Host string
	}

	var cfg Config
	fields := extractFields(reflect.ValueOf(&cfg).Elem(), "", nil)

	err := loadFile(&cfg, "testdata/nonexistent.json", fields)
	if err == nil {
		t.Fatal("loadFile: expected error for missing file")
	}
	if !strings.Contains(err.Error(), "open config file") {
		t.Errorf("error = %q, want it to mention 'open config file'", err.Error())
	}
}

func TestLoadFile_InvalidJSON(t *testing.T) {
	type Config struct {
		Host string
	}

	var cfg Config
	fields := extractFields(reflect.ValueOf(&cfg).Elem(), "", nil)

	err := loadFile(&cfg, "testdata/invalid.json", fields)
	if err == nil {
		t.Fatal("loadFile: expected error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "decode") {
		t.Errorf("error = %q, want it to mention 'decode'", err.Error())
	}
}

func TestLoadFile_UnsupportedFormat(t *testing.T) {
	type Config struct {
		Host string
	}

	var cfg Config
	fields := extractFields(reflect.ValueOf(&cfg).Elem(), "", nil)

	err := loadFile(&cfg, "testdata/config.xml", fields)
	if err == nil {
		t.Fatal("loadFile: expected error for unsupported format")
	}
	if !strings.Contains(err.Error(), "unsupported") {
		t.Errorf("error = %q, want it to mention 'unsupported'", err.Error())
	}
}
