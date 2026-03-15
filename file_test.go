package gonfig

import (
	"math"
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
		{"int from non-integral float64", reflect.TypeFor[int](), float64(3.9), nil, true},
		{"int64 from non-integral float64", reflect.TypeFor[int64](), float64(99.5), nil, true},
		{"int from +Inf", reflect.TypeFor[int](), math.Inf(1), nil, true},
		{"int from -Inf", reflect.TypeFor[int](), math.Inf(-1), nil, true},
		{"int from NaN", reflect.TypeFor[int](), math.NaN(), nil, true},
		{"int64 from +Inf", reflect.TypeFor[int64](), math.Inf(1), nil, true},
		{"int from overflow", reflect.TypeFor[int](), float64(1e20), nil, true},
		{"int64 from overflow", reflect.TypeFor[int64](), float64(1e20), nil, true},
		{"int64 from negative overflow", reflect.TypeFor[int64](), float64(-1e20), nil, true},
		{"int64 from boundary overflow 2^63", reflect.TypeFor[int64](), math.Exp2(63), nil, true},
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

func TestDecodeYAML(t *testing.T) {
	input := "host: localhost\nport: 8080\ndebug: true\n"
	data, err := decodeYAML(strings.NewReader(input))
	if err != nil {
		t.Fatalf("decodeYAML: unexpected error: %v", err)
	}
	if data["host"] != "localhost" {
		t.Errorf("host = %v, want localhost", data["host"])
	}
	// YAML decodes integers as int, not float64.
	if data["port"] != 8080 {
		t.Errorf("port = %v (%T), want 8080", data["port"], data["port"])
	}
	if data["debug"] != true {
		t.Errorf("debug = %v, want true", data["debug"])
	}
}

func TestDecodeYAML_Invalid(t *testing.T) {
	_, err := decodeYAML(strings.NewReader(":\n  :\n  - ][invalid"))
	if err == nil {
		t.Fatal("decodeYAML: expected error for invalid YAML")
	}
}

func TestLoadFile_YAML_Flat(t *testing.T) {
	type Config struct {
		Host     string  `gonfig:"host"`
		Port     int     `gonfig:"port"`
		Debug    bool    `gonfig:"debug"`
		LogLevel string  `gonfig:"log_level"`
		Rate     float64 `gonfig:"rate"`
	}

	var cfg Config
	fields := extractFields(reflect.ValueOf(&cfg).Elem(), "", nil)

	if err := loadFile(&cfg, "testdata/valid.yaml", fields); err != nil {
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

func TestLoadFile_YAML_Nested(t *testing.T) {
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

	if err := loadFile(&cfg, "testdata/nested.yaml", fields); err != nil {
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

func TestLoadFile_YAML_Invalid(t *testing.T) {
	type Config struct {
		Host string
	}

	var cfg Config
	fields := extractFields(reflect.ValueOf(&cfg).Elem(), "", nil)

	err := loadFile(&cfg, "testdata/invalid.yaml", fields)
	if err == nil {
		t.Fatal("loadFile: expected error for invalid YAML")
	}
	if !strings.Contains(err.Error(), "decode") {
		t.Errorf("error = %q, want it to mention 'decode'", err.Error())
	}
}

func TestDecodeTOML(t *testing.T) {
	input := "host = \"localhost\"\nport = 8080\ndebug = true\n"
	data, err := decodeTOML(strings.NewReader(input))
	if err != nil {
		t.Fatalf("decodeTOML: unexpected error: %v", err)
	}
	if data["host"] != "localhost" {
		t.Errorf("host = %v, want localhost", data["host"])
	}
	// TOML decodes integers as int64.
	if data["port"] != int64(8080) {
		t.Errorf("port = %v (%T), want 8080", data["port"], data["port"])
	}
	if data["debug"] != true {
		t.Errorf("debug = %v, want true", data["debug"])
	}
}

func TestDecodeTOML_Invalid(t *testing.T) {
	_, err := decodeTOML(strings.NewReader("[invalid\nkey = "))
	if err == nil {
		t.Fatal("decodeTOML: expected error for invalid TOML")
	}
}

func TestLoadFile_TOML_Flat(t *testing.T) {
	type Config struct {
		Host     string  `gonfig:"host"`
		Port     int     `gonfig:"port"`
		Debug    bool    `gonfig:"debug"`
		LogLevel string  `gonfig:"log_level"`
		Rate     float64 `gonfig:"rate"`
	}

	var cfg Config
	fields := extractFields(reflect.ValueOf(&cfg).Elem(), "", nil)

	if err := loadFile(&cfg, "testdata/valid.toml", fields); err != nil {
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

func TestLoadFile_TOML_Nested(t *testing.T) {
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

	if err := loadFile(&cfg, "testdata/nested.toml", fields); err != nil {
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

func TestLoadFile_TOML_Invalid(t *testing.T) {
	type Config struct {
		Host string
	}

	var cfg Config
	fields := extractFields(reflect.ValueOf(&cfg).Elem(), "", nil)

	err := loadFile(&cfg, "testdata/invalid.toml", fields)
	if err == nil {
		t.Fatal("loadFile: expected error for invalid TOML")
	}
	if !strings.Contains(err.Error(), "decode") {
		t.Errorf("error = %q, want it to mention 'decode'", err.Error())
	}
}

// --- Slice tests (file sources with native arrays) ---

func TestApplyMap_SliceFields(t *testing.T) {
	type Config struct {
		Tags  []string  `gonfig:"tags"`
		Ports []int     `gonfig:"ports"`
		Rates []float64 `gonfig:"rates"`
	}

	var cfg Config
	fields := extractFields(reflect.ValueOf(&cfg).Elem(), "", nil)

	data := map[string]any{
		"tags":  []any{"web", "api", "v2"},
		"ports": []any{float64(8080), float64(8443), float64(9090)},
		"rates": []any{float64(1.5), float64(2.7), float64(3.14)},
	}

	if err := applyMap(&cfg, data, fields); err != nil {
		t.Fatalf("applyMap: unexpected error: %v", err)
	}

	wantTags := []string{"web", "api", "v2"}
	if !reflect.DeepEqual(cfg.Tags, wantTags) {
		t.Errorf("Tags = %v, want %v", cfg.Tags, wantTags)
	}
	wantPorts := []int{8080, 8443, 9090}
	if !reflect.DeepEqual(cfg.Ports, wantPorts) {
		t.Errorf("Ports = %v, want %v", cfg.Ports, wantPorts)
	}
	wantRates := []float64{1.5, 2.7, 3.14}
	if !reflect.DeepEqual(cfg.Rates, wantRates) {
		t.Errorf("Rates = %v, want %v", cfg.Rates, wantRates)
	}
}

func TestLoadFile_JSON_Slices(t *testing.T) {
	type Config struct {
		Tags  []string  `gonfig:"tags"`
		Ports []int     `gonfig:"ports"`
		Rates []float64 `gonfig:"rates"`
	}

	var cfg Config
	fields := extractFields(reflect.ValueOf(&cfg).Elem(), "", nil)

	if err := loadFile(&cfg, "testdata/slices.json", fields); err != nil {
		t.Fatalf("loadFile: unexpected error: %v", err)
	}

	if !reflect.DeepEqual(cfg.Tags, []string{"web", "api", "v2"}) {
		t.Errorf("Tags = %v", cfg.Tags)
	}
	if !reflect.DeepEqual(cfg.Ports, []int{8080, 8443, 9090}) {
		t.Errorf("Ports = %v", cfg.Ports)
	}
	if !reflect.DeepEqual(cfg.Rates, []float64{1.5, 2.7, 3.14}) {
		t.Errorf("Rates = %v", cfg.Rates)
	}
}

func TestLoadFile_YAML_Slices(t *testing.T) {
	type Config struct {
		Tags  []string  `gonfig:"tags"`
		Ports []int     `gonfig:"ports"`
		Rates []float64 `gonfig:"rates"`
	}

	var cfg Config
	fields := extractFields(reflect.ValueOf(&cfg).Elem(), "", nil)

	if err := loadFile(&cfg, "testdata/slices.yaml", fields); err != nil {
		t.Fatalf("loadFile: unexpected error: %v", err)
	}

	if !reflect.DeepEqual(cfg.Tags, []string{"web", "api", "v2"}) {
		t.Errorf("Tags = %v", cfg.Tags)
	}
	if !reflect.DeepEqual(cfg.Ports, []int{8080, 8443, 9090}) {
		t.Errorf("Ports = %v", cfg.Ports)
	}
	if !reflect.DeepEqual(cfg.Rates, []float64{1.5, 2.7, 3.14}) {
		t.Errorf("Rates = %v", cfg.Rates)
	}
}

func TestLoadFile_TOML_Slices(t *testing.T) {
	type Config struct {
		Tags  []string  `gonfig:"tags"`
		Ports []int     `gonfig:"ports"`
		Rates []float64 `gonfig:"rates"`
	}

	var cfg Config
	fields := extractFields(reflect.ValueOf(&cfg).Elem(), "", nil)

	if err := loadFile(&cfg, "testdata/slices.toml", fields); err != nil {
		t.Fatalf("loadFile: unexpected error: %v", err)
	}

	if !reflect.DeepEqual(cfg.Tags, []string{"web", "api", "v2"}) {
		t.Errorf("Tags = %v", cfg.Tags)
	}
	if !reflect.DeepEqual(cfg.Ports, []int{8080, 8443, 9090}) {
		t.Errorf("Ports = %v", cfg.Ports)
	}
	if !reflect.DeepEqual(cfg.Rates, []float64{1.5, 2.7, 3.14}) {
		t.Errorf("Rates = %v", cfg.Rates)
	}
}

// --- Map tests (file sources with native maps) ---

func TestApplyMap_MapStringString(t *testing.T) {
	type Config struct {
		Labels map[string]string `gonfig:"labels"`
	}

	var cfg Config
	fields := extractFields(reflect.ValueOf(&cfg).Elem(), "", nil)

	data := map[string]any{
		"labels": map[string]any{
			"env":    "production",
			"region": "us-east-1",
			"team":   "platform",
		},
	}

	if err := applyMap(&cfg, data, fields); err != nil {
		t.Fatalf("applyMap: unexpected error: %v", err)
	}

	want := map[string]string{"env": "production", "region": "us-east-1", "team": "platform"}
	if !reflect.DeepEqual(cfg.Labels, want) {
		t.Errorf("Labels = %v, want %v", cfg.Labels, want)
	}
}

func TestApplyMap_MapStringAny(t *testing.T) {
	type Config struct {
		Metadata map[string]any `gonfig:"metadata"`
	}

	var cfg Config
	fields := extractFields(reflect.ValueOf(&cfg).Elem(), "", nil)

	data := map[string]any{
		"metadata": map[string]any{
			"version": "1.0",
			"count":   float64(42),
			"enabled": true,
		},
	}

	if err := applyMap(&cfg, data, fields); err != nil {
		t.Fatalf("applyMap: unexpected error: %v", err)
	}

	if cfg.Metadata["version"] != "1.0" {
		t.Errorf("Metadata[version] = %v", cfg.Metadata["version"])
	}
	if cfg.Metadata["count"] != float64(42) {
		t.Errorf("Metadata[count] = %v", cfg.Metadata["count"])
	}
	if cfg.Metadata["enabled"] != true {
		t.Errorf("Metadata[enabled] = %v", cfg.Metadata["enabled"])
	}
}

func TestLoadFile_JSON_Maps(t *testing.T) {
	type Config struct {
		Labels   map[string]string `gonfig:"labels"`
		Metadata map[string]any    `gonfig:"metadata"`
	}

	var cfg Config
	fields := extractFields(reflect.ValueOf(&cfg).Elem(), "", nil)

	if err := loadFile(&cfg, "testdata/maps.json", fields); err != nil {
		t.Fatalf("loadFile: unexpected error: %v", err)
	}

	wantLabels := map[string]string{"env": "production", "region": "us-east-1", "team": "platform"}
	if !reflect.DeepEqual(cfg.Labels, wantLabels) {
		t.Errorf("Labels = %v, want %v", cfg.Labels, wantLabels)
	}
	if cfg.Metadata["version"] != "1.0" {
		t.Errorf("Metadata[version] = %v", cfg.Metadata["version"])
	}
}

func TestLoadFile_YAML_Maps(t *testing.T) {
	type Config struct {
		Labels   map[string]string `gonfig:"labels"`
		Metadata map[string]any    `gonfig:"metadata"`
	}

	var cfg Config
	fields := extractFields(reflect.ValueOf(&cfg).Elem(), "", nil)

	if err := loadFile(&cfg, "testdata/maps.yaml", fields); err != nil {
		t.Fatalf("loadFile: unexpected error: %v", err)
	}

	wantLabels := map[string]string{"env": "production", "region": "us-east-1", "team": "platform"}
	if !reflect.DeepEqual(cfg.Labels, wantLabels) {
		t.Errorf("Labels = %v, want %v", cfg.Labels, wantLabels)
	}
	if cfg.Metadata["version"] != "1.0" {
		t.Errorf("Metadata[version] = %v", cfg.Metadata["version"])
	}
}

func TestLoadFile_TOML_Maps(t *testing.T) {
	type Config struct {
		Labels   map[string]string `gonfig:"labels"`
		Metadata map[string]any    `gonfig:"metadata"`
	}

	var cfg Config
	fields := extractFields(reflect.ValueOf(&cfg).Elem(), "", nil)

	if err := loadFile(&cfg, "testdata/maps.toml", fields); err != nil {
		t.Fatalf("loadFile: unexpected error: %v", err)
	}

	wantLabels := map[string]string{"env": "production", "region": "us-east-1", "team": "platform"}
	if !reflect.DeepEqual(cfg.Labels, wantLabels) {
		t.Errorf("Labels = %v, want %v", cfg.Labels, wantLabels)
	}
	if cfg.Metadata["version"] != "1.0" {
		t.Errorf("Metadata[version] = %v", cfg.Metadata["version"])
	}
}

// --- Slice edge cases ---

func TestApplyMap_EmptySlice(t *testing.T) {
	type Config struct {
		Tags []string `gonfig:"tags"`
	}

	var cfg Config
	fields := extractFields(reflect.ValueOf(&cfg).Elem(), "", nil)

	data := map[string]any{
		"tags": []any{},
	}

	if err := applyMap(&cfg, data, fields); err != nil {
		t.Fatalf("applyMap: unexpected error: %v", err)
	}

	if cfg.Tags == nil || len(cfg.Tags) != 0 {
		t.Errorf("Tags = %v, want empty non-nil slice", cfg.Tags)
	}
}

func TestApplyMap_SingleElementSlice(t *testing.T) {
	type Config struct {
		Tags  []string `gonfig:"tags"`
		Ports []int    `gonfig:"ports"`
	}

	var cfg Config
	fields := extractFields(reflect.ValueOf(&cfg).Elem(), "", nil)

	data := map[string]any{
		"tags":  []any{"single"},
		"ports": []any{float64(8080)},
	}

	if err := applyMap(&cfg, data, fields); err != nil {
		t.Fatalf("applyMap: unexpected error: %v", err)
	}

	if !reflect.DeepEqual(cfg.Tags, []string{"single"}) {
		t.Errorf("Tags = %v, want [single]", cfg.Tags)
	}
	if !reflect.DeepEqual(cfg.Ports, []int{8080}) {
		t.Errorf("Ports = %v, want [8080]", cfg.Ports)
	}
}

func TestSetSliceFromAny_IntNonIntegralFloat64(t *testing.T) {
	field := reflect.New(reflect.TypeFor[[]int]()).Elem()
	err := setSliceFromAny(field, []any{float64(1), float64(2.5)}, field.Type())
	if err == nil {
		t.Fatal("expected error for non-integral float64 in int slice")
	}
}

func TestSetSliceFromAny_IntNonFiniteFloat64(t *testing.T) {
	tests := []struct {
		name string
		val  float64
	}{
		{"+Inf", math.Inf(1)},
		{"-Inf", math.Inf(-1)},
		{"NaN", math.NaN()},
		{"overflow", 1e20},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field := reflect.New(reflect.TypeFor[[]int]()).Elem()
			err := setSliceFromAny(field, []any{tt.val}, field.Type())
			if err == nil {
				t.Fatalf("expected error for %s in int slice", tt.name)
			}
		})
	}
}

func TestSetSliceFromAny_Float64(t *testing.T) {
	field := reflect.New(reflect.TypeFor[[]float64]()).Elem()
	err := setSliceFromAny(field, []any{float64(1.5), float64(2.7)}, field.Type())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := field.Interface().([]float64)
	want := []float64{1.5, 2.7}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestSetSliceFromAny_IntFromYAML(t *testing.T) {
	// YAML decodes integers as int, not float64
	field := reflect.New(reflect.TypeFor[[]int]()).Elem()
	err := setSliceFromAny(field, []any{1, 2, 3}, field.Type())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := field.Interface().([]int)
	want := []int{1, 2, 3}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestSetSliceFromAny_Float64FromYAMLInt(t *testing.T) {
	// YAML decodes integers as int; []float64 field should accept them
	field := reflect.New(reflect.TypeFor[[]float64]()).Elem()
	err := setSliceFromAny(field, []any{1, 2, 3}, field.Type())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := field.Interface().([]float64)
	want := []float64{1, 2, 3}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestSetMapFromAny_NotAMap(t *testing.T) {
	field := reflect.New(reflect.TypeFor[map[string]string]()).Elem()
	err := setMapFromAny(field, "not a map", field.Type())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSetMapFromAny_StringValueMismatch(t *testing.T) {
	field := reflect.New(reflect.TypeFor[map[string]string]()).Elem()
	err := setMapFromAny(field, map[string]any{"key": 42}, field.Type())
	if err == nil {
		t.Fatal("expected error for non-string value in map[string]string")
	}
}

// --- Slice env/flag tests ---

func TestSlice_EnvCommaSeparated(t *testing.T) {
	type Config struct {
		Tags  []string  `gonfig:"tags"`
		Ports []int     `gonfig:"ports"`
		Rates []float64 `gonfig:"rates"`
	}

	var cfg Config
	fields := extractFields(reflect.ValueOf(&cfg).Elem(), "", nil)

	t.Setenv("TAGS", "web,api,v2")
	t.Setenv("PORTS", "8080,8443,9090")
	t.Setenv("RATES", "1.5,2.7,3.14")

	if err := applyEnv(&cfg, fields, ""); err != nil {
		t.Fatalf("applyEnv: unexpected error: %v", err)
	}

	if !reflect.DeepEqual(cfg.Tags, []string{"web", "api", "v2"}) {
		t.Errorf("Tags = %v", cfg.Tags)
	}
	if !reflect.DeepEqual(cfg.Ports, []int{8080, 8443, 9090}) {
		t.Errorf("Ports = %v", cfg.Ports)
	}
	if !reflect.DeepEqual(cfg.Rates, []float64{1.5, 2.7, 3.14}) {
		t.Errorf("Rates = %v", cfg.Rates)
	}
}

func TestSlice_FlagCommaSeparated(t *testing.T) {
	type Config struct {
		Tags  []string  `gonfig:"tags"`
		Ports []int     `gonfig:"ports"`
		Rates []float64 `gonfig:"rates"`
	}

	var cfg Config
	fields := extractFields(reflect.ValueOf(&cfg).Elem(), "", nil)

	args := []string{"--tags", "web,api,v2", "--ports", "8080,8443,9090", "--rates", "1.5,2.7,3.14"}

	if err := applyFlags(&cfg, fields, args); err != nil {
		t.Fatalf("applyFlags: unexpected error: %v", err)
	}

	if !reflect.DeepEqual(cfg.Tags, []string{"web", "api", "v2"}) {
		t.Errorf("Tags = %v", cfg.Tags)
	}
	if !reflect.DeepEqual(cfg.Ports, []int{8080, 8443, 9090}) {
		t.Errorf("Ports = %v", cfg.Ports)
	}
	if !reflect.DeepEqual(cfg.Rates, []float64{1.5, 2.7, 3.14}) {
		t.Errorf("Rates = %v", cfg.Rates)
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
