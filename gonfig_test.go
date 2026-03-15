package gonfig

import "testing"

func TestLoad_Stub(t *testing.T) {
	type Config struct {
		Host string
	}
	var cfg Config
	err := Load(&cfg)
	if err != nil {
		t.Fatalf("Load() returned unexpected error: %v", err)
	}
}
