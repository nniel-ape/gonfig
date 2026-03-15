// Package gonfig provides multi-source configuration loading for Go applications.
//
// It loads configuration from multiple sources (environment variables, command-line flags,
// YAML, TOML, JSON files) into a struct using Go struct tags, with a clear priority order:
// flag > env > file > default.
package gonfig

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"reflect"
)

// Sentinel errors returned by Load.
var (
	// ErrInvalidTarget is returned when Load is called with a non-pointer or non-struct target.
	ErrInvalidTarget = errors.New("target must be a non-nil pointer to a struct")

	// ErrFileNotFound is returned when a specified config file does not exist.
	ErrFileNotFound = errors.New("config file not found")

	// ErrParse is returned when type conversion fails during config loading.
	ErrParse = errors.New("parse error")

	// ErrValidation is returned when validation fails after config loading.
	ErrValidation = errors.New("validation error")
)

// Format specifies a config file format for use with WithFileContent.
type Format string

const (
	JSON Format = "json"
	YAML Format = "yaml"
	TOML Format = "toml"
)

// Option configures the behavior of Load.
type Option func(*options)

type options struct {
	envPrefix   string
	filePaths   []string
	fileContent []fileContentSource
	flagArgs    []string
	hasFlags    bool
}

type fileContentSource struct {
	data   []byte
	format Format
}

// WithEnvPrefix sets a prefix for environment variable lookups.
// For example, WithEnvPrefix("APP") causes the field DB.Host to be read from APP_DB_HOST.
func WithEnvPrefix(prefix string) Option {
	return func(o *options) {
		o.envPrefix = prefix
	}
}

// WithFile adds a config file path to load. The format is detected from the file extension.
// Multiple files can be specified; they are loaded in order (later files override earlier ones).
func WithFile(path string) Option {
	return func(o *options) {
		o.filePaths = append(o.filePaths, path)
	}
}

// WithFlags sets command-line arguments to parse as flags.
func WithFlags(args []string) Option {
	return func(o *options) {
		o.flagArgs = args
		o.hasFlags = true
	}
}

// WithFileContent provides config file content directly as bytes with a specified format.
// This is useful for testing or embedding config data.
func WithFileContent(data []byte, format Format) Option {
	return func(o *options) {
		o.fileContent = append(o.fileContent, fileContentSource{data: data, format: format})
	}
}

// Load populates the target struct with configuration values from the configured sources.
// The target must be a non-nil pointer to a struct.
//
// Sources are applied in priority order (lowest to highest):
// defaults → file → env → flags
//
// Later sources override values set by earlier sources.
func Load(target any, opts ...Option) error {
	if target == nil {
		return ErrInvalidTarget
	}

	rv := reflect.ValueOf(target)
	if rv.Kind() != reflect.Pointer || rv.Elem().Kind() != reflect.Struct {
		return ErrInvalidTarget
	}

	var o options
	for _, opt := range opts {
		opt(&o)
	}

	fields := extractFields(rv.Elem(), "", nil)

	// 1. Apply defaults.
	if err := applyDefaults(target, fields); err != nil {
		return fmt.Errorf("%w: %v", ErrParse, err)
	}

	// 2. Apply file sources.
	for _, path := range o.filePaths {
		if err := loadFile(target, path, fields); err != nil {
			if isFileNotFound(err) {
				return fmt.Errorf("%w: %v", ErrFileNotFound, err)
			}
			return fmt.Errorf("%w: %v", ErrParse, err)
		}
	}

	// 2b. Apply inline file content sources.
	for _, fc := range o.fileContent {
		if err := loadFileContent(target, fc.data, fc.format, fields); err != nil {
			return fmt.Errorf("%w: %v", ErrParse, err)
		}
	}

	// 3. Apply environment variables.
	if err := applyEnv(target, fields, o.envPrefix); err != nil {
		return fmt.Errorf("%w: %v", ErrParse, err)
	}

	// 4. Apply flags.
	if o.hasFlags {
		if err := applyFlags(target, fields, o.flagArgs); err != nil {
			return fmt.Errorf("%w: %v", ErrParse, err)
		}
	}

	return nil
}

// loadFileContent decodes config from raw bytes with the given format and applies it to target.
func loadFileContent(target any, data []byte, format Format, fields []fieldInfo) error {
	r := bytes.NewReader(data)
	var (
		m   map[string]any
		err error
	)

	switch format {
	case JSON:
		m, err = decodeJSON(r)
	case YAML:
		m, err = decodeYAML(r)
	case TOML:
		m, err = decodeTOML(r)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
	if err != nil {
		return fmt.Errorf("decode %s: %w", format, err)
	}

	return applyMap(target, m, fields)
}

// isFileNotFound checks if an error is caused by a missing file.
func isFileNotFound(err error) bool {
	var pathErr *os.PathError
	if errors.As(err, &pathErr) {
		return errors.Is(pathErr.Err, os.ErrNotExist)
	}
	return false
}
