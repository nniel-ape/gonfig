// Package gonfig provides multi-source configuration loading for Go applications.
//
// It loads configuration from multiple sources (environment variables, command-line flags,
// YAML, TOML, JSON files) into a struct using Go struct tags, with a clear priority order:
// flag > env > file > default.
package gonfig

// Load populates the target struct with configuration values from the configured sources.
func Load(target any, opts ...Option) error {
	return nil
}

// Option configures the behavior of Load.
type Option func(*options)

type options struct{}
