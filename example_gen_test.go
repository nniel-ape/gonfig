package gonfig

import (
	"strings"
	"testing"
	"time"
)

func TestExample_FlatStruct_YAML(t *testing.T) {
	type Config struct {
		Host    string  `default:"localhost" description:"server host" validate:"required"`
		Port    int     `default:"8080"      description:"server port" validate:"min=1,max=65535"`
		Rate    float64 `default:"1.5"       description:"request rate"`
		Verbose bool    `default:"true"      description:"verbose output"`
		Timeout string  `default:"30s"       description:"request timeout"`
	}

	var cfg Config
	got := Example(&cfg, YAML)

	assertContains(t, got, `# server host (required)`)
	assertContains(t, got, `host: "localhost"`)
	assertContains(t, got, `# server port (min=1,max=65535)`)
	assertContains(t, got, `port: 8080`)
	assertContains(t, got, `rate: 1.5`)
	assertContains(t, got, `verbose: true`)
	assertContains(t, got, `timeout: "30s"`)
}

func TestExample_FlatStruct_JSON(t *testing.T) {
	type Config struct {
		Host string `default:"localhost"`
		Port int    `default:"8080"`
	}

	var cfg Config
	got := Example(&cfg, JSON)

	// JSON has no comments.
	if strings.Contains(got, "#") {
		t.Error("JSON output should not contain comments")
	}
	assertContains(t, got, `"host": "localhost"`)
	assertContains(t, got, `"port": 8080`)
}

func TestExample_FlatStruct_TOML(t *testing.T) {
	type Config struct {
		Host    string `default:"localhost" description:"server host" validate:"required"`
		Port    int    `default:"8080"      description:"server port"`
		Verbose bool   `default:"true"      description:"verbose"`
	}

	var cfg Config
	got := Example(&cfg, TOML)

	assertContains(t, got, `# server host (required)`)
	assertContains(t, got, `host = "localhost"`)
	assertContains(t, got, `# server port`)
	assertContains(t, got, `port = 8080`)
	assertContains(t, got, `verbose = true`)
}

func TestExample_NestedStruct_YAML(t *testing.T) {
	type Config struct {
		DB struct {
			Host string `default:"localhost" description:"database host"`
			Port int    `default:"5432"      description:"database port"`
		}
		LogLevel string `default:"info" description:"logging level"`
	}

	var cfg Config
	got := Example(&cfg, YAML)

	assertContains(t, got, `db:`)
	assertContains(t, got, `  host: "localhost"`)
	assertContains(t, got, `  port: 5432`)
	assertContains(t, got, `log_level: "info"`)
}

func TestExample_NestedStruct_TOML(t *testing.T) {
	type Config struct {
		DB struct {
			Host string `default:"localhost" description:"database host"`
			Port int    `default:"5432"      description:"database port"`
		}
		LogLevel string `default:"info" description:"logging level"`
	}

	var cfg Config
	got := Example(&cfg, TOML)

	assertContains(t, got, `[db]`)
	assertContains(t, got, `host = "localhost"`)
	assertContains(t, got, `port = 5432`)
	assertContains(t, got, `log_level = "info"`)
}

func TestExample_NestedStruct_JSON(t *testing.T) {
	type Config struct {
		DB struct {
			Host string `default:"localhost"`
			Port int    `default:"5432"`
		}
		LogLevel string `default:"info"`
	}

	var cfg Config
	got := Example(&cfg, JSON)

	assertContains(t, got, `"db": {`)
	assertContains(t, got, `"host": "localhost"`)
	assertContains(t, got, `"port": 5432`)
	assertContains(t, got, `"log_level": "info"`)
}

func TestExample_FieldsWithoutDefaults(t *testing.T) {
	type Config struct {
		Name    string `description:"app name" validate:"required"`
		Count   int    `description:"item count"`
		Enabled bool   `description:"enabled flag"`
	}

	var cfg Config
	got := Example(&cfg, YAML)

	assertContains(t, got, `# app name (required)`)
	assertContains(t, got, `name: ""`)
	assertContains(t, got, `count: 0`)
	assertContains(t, got, `enabled: false`)
}

func TestExample_Slices(t *testing.T) {
	type Config struct {
		Tags  []string  `default:"web,api" description:"tags"`
		Ports []int     `default:"80,443"  description:"ports"`
		Rates []float64 `default:"1.0,2.5" description:"rates"`
		Empty []string  `description:"empty slice"`
	}

	var cfg Config
	got := Example(&cfg, YAML)

	assertContains(t, got, `tags: ["web", "api"]`)
	assertContains(t, got, `ports: [80, 443]`)
	assertContains(t, got, `rates: [1, 2.5]`)
	assertContains(t, got, `empty: []`)
}

func TestExample_Maps(t *testing.T) {
	type Config struct {
		Labels   map[string]string `description:"labels"`
		Metadata map[string]any    `description:"metadata"`
	}

	var cfg Config
	got := Example(&cfg, YAML)

	assertContains(t, got, `labels: {}`)
	assertContains(t, got, `metadata: {}`)
}

func TestExample_Duration(t *testing.T) {
	type Config struct {
		Timeout time.Duration `default:"5s" description:"request timeout"`
		ZeroDur time.Duration `description:"zero timeout"`
	}

	var cfg Config
	got := Example(&cfg, YAML)

	assertContains(t, got, `timeout: "5s"`)
	assertContains(t, got, `zero_dur: "0s"`)
}

func TestExample_EmptyStruct(t *testing.T) {
	type Config struct{}

	var cfg Config
	got := Example(&cfg, YAML)

	if got != "" {
		t.Errorf("empty struct should produce empty output, got: %q", got)
	}
}

func TestExample_NilTarget(t *testing.T) {
	got := Example(nil, YAML)
	if got != "" {
		t.Errorf("nil target should produce empty string, got: %q", got)
	}
}

func TestExample_NonStructTarget(t *testing.T) {
	var s string
	got := Example(&s, YAML)
	if got != "" {
		t.Errorf("non-struct target should produce empty string, got: %q", got)
	}
}

func TestExample_GonfigTagOverride(t *testing.T) {
	type Strategy struct {
		Name   string  `default:"momentum"`
		Weight float64 `default:"0.5"`
	}
	type Config struct {
		Strategy Strategy `gonfig:"lm"`
	}

	var cfg Config
	got := Example(&cfg, YAML)

	assertContains(t, got, `lm:`)
	assertContains(t, got, `  name: "momentum"`)
	assertContains(t, got, `  weight: 0.5`)
}

func TestExample_GonfigTagOverride_TOML(t *testing.T) {
	type Strategy struct {
		Name string `default:"momentum"`
	}
	type Config struct {
		Strategy Strategy `gonfig:"lm"`
	}

	var cfg Config
	got := Example(&cfg, TOML)

	assertContains(t, got, `[lm]`)
	assertContains(t, got, `name = "momentum"`)
}

func TestExtractAndRemoveGenerateConfig_EqualsSyntax(t *testing.T) {
	format, args := extractAndRemoveGenerateConfig([]string{"--port", "8080", "--generate-config=yaml"})
	if format != "yaml" {
		t.Errorf("format = %q, want %q", format, "yaml")
	}
	if len(args) != 2 || args[0] != "--port" || args[1] != "8080" {
		t.Errorf("args = %v, want [--port 8080]", args)
	}
}

func TestExtractAndRemoveGenerateConfig_SpaceSyntax(t *testing.T) {
	format, args := extractAndRemoveGenerateConfig([]string{"--generate-config", "json", "--port", "8080"})
	if format != "json" {
		t.Errorf("format = %q, want %q", format, "json")
	}
	if len(args) != 2 || args[0] != "--port" || args[1] != "8080" {
		t.Errorf("args = %v, want [--port 8080]", args)
	}
}

func TestExtractAndRemoveGenerateConfig_NotPresent(t *testing.T) {
	format, args := extractAndRemoveGenerateConfig([]string{"--port", "8080"})
	if format != "" {
		t.Errorf("format = %q, want empty", format)
	}
	if len(args) != 2 {
		t.Errorf("args = %v, want [--port 8080]", args)
	}
}

func assertContains(t *testing.T, haystack, needle string) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Errorf("expected output to contain %q\nfull output:\n%s", needle, haystack)
	}
}
