package gonfig

import (
	"reflect"
	"testing"
	"time"
)

func TestApplyEnv_BasicTypes(t *testing.T) {
	type Config struct {
		Host    string
		Port    int
		Debug   bool
		Rate    float64
		Timeout time.Duration
	}

	tests := []struct {
		name string
		envs map[string]string
		want Config
	}{
		{
			name: "string value",
			envs: map[string]string{"HOST": "localhost"},
			want: Config{Host: "localhost"},
		},
		{
			name: "int value",
			envs: map[string]string{"PORT": "8080"},
			want: Config{Port: 8080},
		},
		{
			name: "bool value",
			envs: map[string]string{"DEBUG": "true"},
			want: Config{Debug: true},
		},
		{
			name: "float value",
			envs: map[string]string{"RATE": "3.14"},
			want: Config{Rate: 3.14},
		},
		{
			name: "duration value",
			envs: map[string]string{"TIMEOUT": "5s"},
			want: Config{Timeout: 5 * time.Second},
		},
		{
			name: "multiple values",
			envs: map[string]string{
				"HOST":  "remotehost",
				"PORT":  "9090",
				"DEBUG": "true",
			},
			want: Config{Host: "remotehost", Port: 9090, Debug: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.envs {
				t.Setenv(k, v)
			}

			var cfg Config
			v := reflect.ValueOf(&cfg).Elem()
			fields := extractFields(v, "", nil)

			if err := applyEnv(&cfg, fields, ""); err != nil {
				t.Fatalf("applyEnv() error = %v", err)
			}

			if cfg != tt.want {
				t.Errorf("applyEnv() got = %+v, want = %+v", cfg, tt.want)
			}
		})
	}
}

func TestApplyEnv_WithPrefix(t *testing.T) {
	type Config struct {
		Host string
		Port int
	}

	type NestedConfig struct {
		DB struct {
			Host string
			Port int
		}
	}

	t.Run("simple prefix", func(t *testing.T) {
		t.Setenv("APP_HOST", "prefixed-host")
		t.Setenv("APP_PORT", "3000")

		var cfg Config
		v := reflect.ValueOf(&cfg).Elem()
		fields := extractFields(v, "", nil)

		if err := applyEnv(&cfg, fields, "APP"); err != nil {
			t.Fatalf("applyEnv() error = %v", err)
		}

		if cfg.Host != "prefixed-host" {
			t.Errorf("Host = %q, want %q", cfg.Host, "prefixed-host")
		}
		if cfg.Port != 3000 {
			t.Errorf("Port = %d, want %d", cfg.Port, 3000)
		}
	})

	t.Run("prefix with nested struct", func(t *testing.T) {
		t.Setenv("APP_DB_HOST", "db-host")
		t.Setenv("APP_DB_PORT", "5432")

		var cfg NestedConfig
		v := reflect.ValueOf(&cfg).Elem()
		fields := extractFields(v, "", nil)

		if err := applyEnv(&cfg, fields, "APP"); err != nil {
			t.Fatalf("applyEnv() error = %v", err)
		}

		if cfg.DB.Host != "db-host" {
			t.Errorf("DB.Host = %q, want %q", cfg.DB.Host, "db-host")
		}
		if cfg.DB.Port != 5432 {
			t.Errorf("DB.Port = %d, want %d", cfg.DB.Port, 5432)
		}
	})
}

func TestApplyEnv_NotSet(t *testing.T) {
	type Config struct {
		Host string
		Port int
		Flag bool
	}

	// Pre-set values to verify they are not overwritten.
	cfg := Config{
		Host: "original",
		Port: 1234,
		Flag: true,
	}

	v := reflect.ValueOf(&cfg).Elem()
	fields := extractFields(v, "", nil)

	// No env vars set — fields should remain unchanged.
	if err := applyEnv(&cfg, fields, ""); err != nil {
		t.Fatalf("applyEnv() error = %v", err)
	}

	if cfg.Host != "original" {
		t.Errorf("Host = %q, want %q", cfg.Host, "original")
	}
	if cfg.Port != 1234 {
		t.Errorf("Port = %d, want %d", cfg.Port, 1234)
	}
	if cfg.Flag != true {
		t.Errorf("Flag = %v, want true", cfg.Flag)
	}
}

func TestApplyEnv_InvalidValue(t *testing.T) {
	type Config struct {
		Port  int
		Debug bool
		Rate  float64
	}

	tests := []struct {
		name string
		envs map[string]string
	}{
		{
			name: "invalid int",
			envs: map[string]string{"PORT": "not-a-number"},
		},
		{
			name: "invalid bool",
			envs: map[string]string{"DEBUG": "not-a-bool"},
		},
		{
			name: "invalid float",
			envs: map[string]string{"RATE": "not-a-float"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.envs {
				t.Setenv(k, v)
			}

			var cfg Config
			v := reflect.ValueOf(&cfg).Elem()
			fields := extractFields(v, "", nil)

			err := applyEnv(&cfg, fields, "")
			if err == nil {
				t.Fatal("applyEnv() expected error, got nil")
			}
		})
	}
}

func TestApplyEnv_ExplicitEnvTag(t *testing.T) {
	type Config struct {
		Host string `env:"CUSTOM_HOST"`
		Port int    `env:"MY_PORT"`
	}

	t.Setenv("CUSTOM_HOST", "tagged-host")
	t.Setenv("MY_PORT", "7777")

	var cfg Config
	v := reflect.ValueOf(&cfg).Elem()
	fields := extractFields(v, "", nil)

	if err := applyEnv(&cfg, fields, ""); err != nil {
		t.Fatalf("applyEnv() error = %v", err)
	}

	if cfg.Host != "tagged-host" {
		t.Errorf("Host = %q, want %q", cfg.Host, "tagged-host")
	}
	if cfg.Port != 7777 {
		t.Errorf("Port = %d, want %d", cfg.Port, 7777)
	}
}

func TestApplyEnv_ExplicitTagWithPrefix(t *testing.T) {
	type Config struct {
		Host string `env:"CUSTOM_HOST"`
	}

	// With prefix, it should be PREFIX_CUSTOM_HOST.
	t.Setenv("APP_CUSTOM_HOST", "prefixed-tagged")

	var cfg Config
	v := reflect.ValueOf(&cfg).Elem()
	fields := extractFields(v, "", nil)

	if err := applyEnv(&cfg, fields, "APP"); err != nil {
		t.Fatalf("applyEnv() error = %v", err)
	}

	if cfg.Host != "prefixed-tagged" {
		t.Errorf("Host = %q, want %q", cfg.Host, "prefixed-tagged")
	}
}

func TestApplyEnv_PartialSet(t *testing.T) {
	type Config struct {
		Host string
		Port int
		Name string
	}

	cfg := Config{Host: "default-host", Port: 80, Name: "default-name"}

	// Only set one env var — others should remain unchanged.
	t.Setenv("PORT", "9090")

	v := reflect.ValueOf(&cfg).Elem()
	fields := extractFields(v, "", nil)

	if err := applyEnv(&cfg, fields, ""); err != nil {
		t.Fatalf("applyEnv() error = %v", err)
	}

	if cfg.Host != "default-host" {
		t.Errorf("Host = %q, want %q", cfg.Host, "default-host")
	}
	if cfg.Port != 9090 {
		t.Errorf("Port = %d, want %d", cfg.Port, 9090)
	}
	if cfg.Name != "default-name" {
		t.Errorf("Name = %q, want %q", cfg.Name, "default-name")
	}
}
