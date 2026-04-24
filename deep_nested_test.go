package gonfig

import (
	"fmt"
	"testing"
)

func TestDeepNestedStruct(t *testing.T) {
	type Config struct {
		A struct {
			B struct {
				C string `default:"deep" description:"deep field"`
			}
		}
	}

	var cfg Config

	err := Load(&cfg)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	if cfg.A.B.C != "deep" {
		t.Errorf("A.B.C = %q, want %q", cfg.A.B.C, "deep")
	}

	usage := Usage(&cfg, WithEnvPrefix("APP"))
	fmt.Println("USAGE:", usage)

	// Verify env/flag derivation for deeply nested
	t.Setenv("APP_A_B_C", "from-env")

	var cfg2 Config

	err = Load(&cfg2, WithEnvPrefix("APP"))
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	if cfg2.A.B.C != "from-env" {
		t.Errorf("A.B.C = %q, want %q", cfg2.A.B.C, "from-env")
	}
}
