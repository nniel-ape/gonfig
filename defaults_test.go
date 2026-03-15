package gonfig

import (
	"reflect"
	"testing"
	"time"
)

func TestApplyDefaults_BasicTypes(t *testing.T) {
	type Config struct {
		Host    string        `default:"localhost"`
		Port    int           `default:"5432"`
		Debug   bool          `default:"true"`
		Rate    float64       `default:"0.75"`
		Timeout time.Duration `default:"30s"`
	}

	var cfg Config
	fields := extractFields(reflect.ValueOf(cfg), "", nil)

	if err := applyDefaults(&cfg, fields); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Host != "localhost" {
		t.Errorf("Host = %q, want %q", cfg.Host, "localhost")
	}
	if cfg.Port != 5432 {
		t.Errorf("Port = %d, want %d", cfg.Port, 5432)
	}
	if cfg.Debug != true {
		t.Errorf("Debug = %v, want true", cfg.Debug)
	}
	if cfg.Rate != 0.75 {
		t.Errorf("Rate = %f, want 0.75", cfg.Rate)
	}
	if cfg.Timeout != 30*time.Second {
		t.Errorf("Timeout = %v, want 30s", cfg.Timeout)
	}
}

func TestApplyDefaults_MissingDefaultTag(t *testing.T) {
	type Config struct {
		Host string `default:"localhost"`
		Port int    // no default tag
		Name string // no default tag
	}

	cfg := Config{Port: 8080, Name: "original"}
	fields := extractFields(reflect.ValueOf(cfg), "", nil)

	if err := applyDefaults(&cfg, fields); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Host != "localhost" {
		t.Errorf("Host = %q, want %q", cfg.Host, "localhost")
	}
	// Fields without default tag should be unchanged.
	if cfg.Port != 8080 {
		t.Errorf("Port = %d, want 8080 (unchanged)", cfg.Port)
	}
	if cfg.Name != "original" {
		t.Errorf("Name = %q, want %q (unchanged)", cfg.Name, "original")
	}
}

func TestApplyDefaults_InvalidDefaultValue(t *testing.T) {
	type Config struct {
		Port int `default:"not-a-number"`
	}

	var cfg Config
	fields := extractFields(reflect.ValueOf(cfg), "", nil)

	err := applyDefaults(&cfg, fields)
	if err == nil {
		t.Fatal("expected error for invalid default, got nil")
	}
}

func TestApplyDefaults_InvalidBoolDefault(t *testing.T) {
	type Config struct {
		Debug bool `default:"yes"`
	}

	var cfg Config
	fields := extractFields(reflect.ValueOf(cfg), "", nil)

	err := applyDefaults(&cfg, fields)
	if err == nil {
		t.Fatal("expected error for invalid bool default, got nil")
	}
}

func TestApplyDefaults_NestedStruct(t *testing.T) {
	type DB struct {
		Host string `default:"localhost"`
		Port int    `default:"5432"`
	}
	type Config struct {
		DB       DB
		LogLevel string `default:"info"`
	}

	var cfg Config
	fields := extractFields(reflect.ValueOf(cfg), "", nil)

	if err := applyDefaults(&cfg, fields); err != nil {
		t.Fatalf("unexpected error: %v", err)
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
}

func TestApplyDefaults_EmptyStringDefault(t *testing.T) {
	type Config struct {
		Name string `default:""`
	}

	cfg := Config{Name: "should-be-overwritten"}
	fields := extractFields(reflect.ValueOf(cfg), "", nil)

	if err := applyDefaults(&cfg, fields); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// default:"" means the default IS an empty string, so it should be applied.
	if cfg.Name != "" {
		t.Errorf("Name = %q, want empty string", cfg.Name)
	}
}

func TestApplyDefaults_ZeroIntDefault(t *testing.T) {
	type Config struct {
		Count int `default:"0"`
	}

	cfg := Config{Count: 99}
	fields := extractFields(reflect.ValueOf(cfg), "", nil)

	if err := applyDefaults(&cfg, fields); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Count != 0 {
		t.Errorf("Count = %d, want 0", cfg.Count)
	}
}

func TestApplyDefaults_SliceDefault(t *testing.T) {
	type Config struct {
		Tags  []string `default:"a,b,c"`
		Ports []int    `default:"80,443"`
	}

	var cfg Config
	fields := extractFields(reflect.ValueOf(cfg), "", nil)

	if err := applyDefaults(&cfg, fields); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	wantTags := []string{"a", "b", "c"}
	if !reflect.DeepEqual(cfg.Tags, wantTags) {
		t.Errorf("Tags = %v, want %v", cfg.Tags, wantTags)
	}

	wantPorts := []int{80, 443}
	if !reflect.DeepEqual(cfg.Ports, wantPorts) {
		t.Errorf("Ports = %v, want %v", cfg.Ports, wantPorts)
	}
}
