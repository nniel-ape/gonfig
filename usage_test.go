package gonfig

import (
	"strings"
	"testing"
	"time"
)

func TestUsage_FlatStruct(t *testing.T) {
	type Config struct {
		Host     string        `default:"localhost" description:"server host"`
		Port     int           `default:"8080"      description:"server port"`
		Debug    bool          `default:"false"     description:"enable debug mode"`
		Timeout  time.Duration `default:"30s"       description:"request timeout"`
		Rate     float64       `default:"1.5"       description:"rate limit"`
	}

	var cfg Config
	output := Usage(&cfg)

	// Verify all fields are present.
	if !strings.Contains(output, "--host") {
		t.Error("output should contain --host flag")
	}
	if !strings.Contains(output, "--port") {
		t.Error("output should contain --port flag")
	}
	if !strings.Contains(output, "--debug") {
		t.Error("output should contain --debug flag")
	}
	if !strings.Contains(output, "--timeout") {
		t.Error("output should contain --timeout flag")
	}
	if !strings.Contains(output, "--rate") {
		t.Error("output should contain --rate flag")
	}

	// Verify env var names.
	if !strings.Contains(output, "HOST") {
		t.Error("output should contain HOST env var")
	}
	if !strings.Contains(output, "PORT") {
		t.Error("output should contain PORT env var")
	}

	// Verify types.
	if !strings.Contains(output, "string") {
		t.Error("output should contain string type")
	}
	if !strings.Contains(output, "int") {
		t.Error("output should contain int type")
	}
	if !strings.Contains(output, "bool") {
		t.Error("output should contain bool type")
	}
	if !strings.Contains(output, "duration") {
		t.Error("output should contain duration type")
	}
	if !strings.Contains(output, "float") {
		t.Error("output should contain float type")
	}

	// Verify defaults.
	if !strings.Contains(output, "(default: localhost)") {
		t.Error("output should contain default value for host")
	}
	if !strings.Contains(output, "(default: 8080)") {
		t.Error("output should contain default value for port")
	}

	// Verify descriptions.
	if !strings.Contains(output, "server host") {
		t.Error("output should contain description for host")
	}
	if !strings.Contains(output, "enable debug mode") {
		t.Error("output should contain description for debug")
	}

	// Verify aligned columns (each line starts with 2-space indent).
	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")
	for _, line := range lines {
		if line != "" && !strings.HasPrefix(line, "  ") {
			t.Errorf("line should start with 2-space indent: %q", line)
		}
	}
}

func TestUsage_NestedStructWithSectionHeaders(t *testing.T) {
	type Config struct {
		LogLevel string `default:"info" description:"logging level"`
		DB       struct {
			Host string `default:"localhost" description:"database host"`
			Port int    `default:"5432"      description:"database port"`
		}
		Cache struct {
			TTL     int    `default:"300"       description:"cache TTL in seconds"`
			Backend string `default:"redis"     description:"cache backend"`
		}
	}

	var cfg Config
	output := Usage(&cfg)

	// Verify section headers.
	if !strings.Contains(output, "DB:\n") {
		t.Error("output should contain DB section header")
	}
	if !strings.Contains(output, "Cache:\n") {
		t.Error("output should contain Cache section header")
	}

	// Verify DB section comes after root and before/after Cache.
	dbIdx := strings.Index(output, "DB:\n")
	cacheIdx := strings.Index(output, "Cache:\n")
	logIdx := strings.Index(output, "--log-level")

	if logIdx > dbIdx {
		t.Error("root fields (log-level) should come before DB section")
	}
	if dbIdx > cacheIdx {
		t.Error("DB section should come before Cache section")
	}

	// Verify fields within sections.
	if !strings.Contains(output, "--db-host") {
		t.Error("output should contain --db-host flag")
	}
	if !strings.Contains(output, "--cache-ttl") {
		t.Error("output should contain --cache-ttl flag")
	}
	if !strings.Contains(output, "DB_HOST") {
		t.Error("output should contain DB_HOST env var")
	}
	if !strings.Contains(output, "CACHE_TTL") {
		t.Error("output should contain CACHE_TTL env var")
	}
}

func TestUsage_FieldsWithoutDescription(t *testing.T) {
	type Config struct {
		Host    string `default:"localhost" description:"the host"`
		Port    int    `default:"8080"`
		Verbose bool
	}

	var cfg Config
	output := Usage(&cfg)

	// All fields should be listed.
	if !strings.Contains(output, "--host") {
		t.Error("output should contain --host")
	}
	if !strings.Contains(output, "--port") {
		t.Error("output should contain --port")
	}
	if !strings.Contains(output, "--verbose") {
		t.Error("output should contain --verbose")
	}

	// Host should have its description.
	if !strings.Contains(output, "the host") {
		t.Error("output should contain description for host")
	}

	// Port and Verbose should still be in the output even without description.
	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")
	if len(lines) != 3 {
		t.Errorf("expected 3 lines, got %d: %q", len(lines), output)
	}
}

func TestUsage_WithEnvPrefix(t *testing.T) {
	type Config struct {
		DB struct {
			Host string `default:"localhost" description:"database host"`
			Port int    `default:"5432"      description:"database port"`
		}
		LogLevel string `default:"info" description:"log level"`
	}

	var cfg Config
	output := Usage(&cfg, WithEnvPrefix("APP"))

	// Env vars should include the prefix.
	if !strings.Contains(output, "APP_DB_HOST") {
		t.Error("output should contain APP_DB_HOST with prefix")
	}
	if !strings.Contains(output, "APP_DB_PORT") {
		t.Error("output should contain APP_DB_PORT with prefix")
	}
	if !strings.Contains(output, "APP_LOG_LEVEL") {
		t.Error("output should contain APP_LOG_LEVEL with prefix")
	}
}

func TestUsage_NilTarget(t *testing.T) {
	output := Usage(nil)
	if output != "" {
		t.Errorf("Usage(nil) = %q, want empty string", output)
	}
}

func TestUsage_NonStructTarget(t *testing.T) {
	s := "not a struct"
	output := Usage(&s)
	if output != "" {
		t.Errorf("Usage(&string) = %q, want empty string", output)
	}
}

func TestUsage_EmptyStruct(t *testing.T) {
	type Config struct{}
	var cfg Config
	output := Usage(&cfg)
	if output != "" {
		t.Errorf("Usage(empty struct) = %q, want empty string", output)
	}
}

func TestUsage_SliceAndMapTypes(t *testing.T) {
	type Config struct {
		Hosts   []string          `description:"list of hosts"`
		Ports   []int             `description:"list of ports"`
		Labels  map[string]string `description:"key-value labels"`
	}

	var cfg Config
	output := Usage(&cfg)

	if !strings.Contains(output, "[]string") {
		t.Error("output should contain []string type")
	}
	if !strings.Contains(output, "[]int") {
		t.Error("output should contain []int type")
	}
	if !strings.Contains(output, "map[string]string") {
		t.Error("output should contain map[string]string type")
	}
}

func TestUsage_ExplicitTagOverrides(t *testing.T) {
	type Config struct {
		Host string `env:"CUSTOM_HOST" flag:"custom-host" description:"custom host"`
	}

	var cfg Config
	output := Usage(&cfg)

	if !strings.Contains(output, "--custom-host") {
		t.Error("output should use explicit flag name --custom-host")
	}
	if !strings.Contains(output, "CUSTOM_HOST") {
		t.Error("output should use explicit env name CUSTOM_HOST")
	}
}

func TestUsage_ExplicitTagOverrides_WithPrefix(t *testing.T) {
	type Config struct {
		Host string `env:"CUSTOM_HOST" flag:"custom-host" description:"custom host"`
	}

	var cfg Config
	output := Usage(&cfg, WithEnvPrefix("APP"))

	// Prefix should be prepended even to explicit env names.
	if !strings.Contains(output, "APP_CUSTOM_HOST") {
		t.Error("output should prepend prefix to explicit env name: APP_CUSTOM_HOST")
	}
}

func TestUsage_OnlyNestedStructs(t *testing.T) {
	type Config struct {
		Server struct {
			Host string `default:"0.0.0.0" description:"bind address"`
			Port int    `default:"8080"    description:"listen port"`
		}
	}

	var cfg Config
	output := Usage(&cfg)

	// Should have Server section header but no root section.
	if !strings.Contains(output, "Server:\n") {
		t.Error("output should contain Server section header")
	}

	// Fields should be indented under section.
	if !strings.Contains(output, "--server-host") {
		t.Error("output should contain --server-host")
	}
	if !strings.Contains(output, "--server-port") {
		t.Error("output should contain --server-port")
	}
}

func TestUsage_NoDefaultValue(t *testing.T) {
	type Config struct {
		Host string `description:"the host"`
		Port int    `default:"8080" description:"the port"`
	}

	var cfg Config
	output := Usage(&cfg)

	// Host line should not have "(default: ...)" text.
	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")
	for _, line := range lines {
		if strings.Contains(line, "--host") && strings.Contains(line, "(default:") {
			t.Error("host line should not have default value")
		}
	}

	// Port should have default.
	if !strings.Contains(output, "(default: 8080)") {
		t.Error("output should contain default value for port")
	}
}

func TestUsage_ShortFlags(t *testing.T) {
	type Config struct {
		Port  int    `short:"p" default:"8080" description:"server port"`
		Debug bool   `short:"d" description:"enable debug mode"`
		Host  string `default:"localhost" description:"server host"` // no short flag
	}

	var cfg Config
	output := Usage(&cfg)

	// Short flags should display as "-p, --port".
	if !strings.Contains(output, "-p, --port") {
		t.Errorf("output should contain '-p, --port', got:\n%s", output)
	}
	if !strings.Contains(output, "-d, --debug") {
		t.Errorf("output should contain '-d, --debug', got:\n%s", output)
	}

	// Host without short flag should display with alignment padding "    --host".
	if !strings.Contains(output, "    --host") {
		t.Errorf("output should contain '    --host' (padded), got:\n%s", output)
	}
}

func TestUsage_ColumnAlignment(t *testing.T) {
	type Config struct {
		H    string `default:"a" description:"short"`
		LongFieldName string `default:"something-longer" description:"long desc"`
	}

	var cfg Config
	output := Usage(&cfg)

	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}

	// Both lines should have the same column positions for env vars and types.
	// The shorter flag should be padded to match the longer one.
	envIdx0 := strings.Index(lines[0], "H ")
	envIdx1 := strings.Index(lines[1], "LONG_FIELD_NAME")
	if envIdx0 != envIdx1 {
		t.Errorf("env columns are not aligned: line0=%d, line1=%d\nline0: %q\nline1: %q",
			envIdx0, envIdx1, lines[0], lines[1])
	}

	typeIdx0 := strings.Index(lines[0], "string")
	typeIdx1 := strings.Index(lines[1], "string")
	if typeIdx0 != typeIdx1 {
		t.Errorf("type columns are not aligned: line0=%d, line1=%d\nline0: %q\nline1: %q",
			typeIdx0, typeIdx1, lines[0], lines[1])
	}
}
