package gonfig

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"gopkg.in/yaml.v3"
)

// loadFile reads a config file, decodes it into a map, and applies values to the target struct.
// The file format is detected from the extension (.json, .yaml/.yml, .toml).
func loadFile(target any, path string, fields []fieldInfo) error {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".json", ".yaml", ".yml", ".toml":
		// supported
	default:
		return fmt.Errorf("unsupported config file format: %s", ext)
	}

	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open config file: %w", err)
	}
	defer f.Close()

	var data map[string]any
	switch ext {
	case ".json":
		data, err = decodeJSON(f)
	case ".yaml", ".yml":
		data, err = decodeYAML(f)
	case ".toml":
		data, err = decodeTOML(f)
	}
	if err != nil {
		return fmt.Errorf("decode %s: %w", ext, err)
	}

	return applyMap(target, data, fields)
}

// decodeJSON decodes JSON from the reader into a map[string]any.
func decodeJSON(r io.Reader) (map[string]any, error) {
	var data map[string]any
	if err := json.NewDecoder(r).Decode(&data); err != nil {
		return nil, err
	}
	return data, nil
}

// decodeYAML decodes YAML from the reader into a map[string]any.
func decodeYAML(r io.Reader) (map[string]any, error) {
	var data map[string]any
	if err := yaml.NewDecoder(r).Decode(&data); err != nil {
		return nil, err
	}
	return data, nil
}

// decodeTOML decodes TOML from the reader into a map[string]any.
func decodeTOML(r io.Reader) (map[string]any, error) {
	var data map[string]any
	if _, err := toml.NewDecoder(r).Decode(&data); err != nil {
		return nil, err
	}
	return data, nil
}

// applyMap walks a nested map and sets struct field values matched by config key.
// Config keys use dot notation (e.g., "db.host") to navigate nested maps.
func applyMap(target any, data map[string]any, fields []fieldInfo) error {
	v := reflect.ValueOf(target).Elem()

	for _, fi := range fields {
		val, ok := lookupMap(data, fi.ConfigKey)
		if !ok {
			continue
		}

		field := fieldByIndex(v, fi.Index)
		if !field.CanSet() {
			return fmt.Errorf("cannot set field %s", fi.Path)
		}

		if err := setFieldFromAny(field, val); err != nil {
			return fmt.Errorf("file value for %s: %w", fi.Path, err)
		}
	}

	return nil
}

// lookupMap navigates a nested map using a dot-separated key path.
// For example, lookupMap(m, "db.host") returns m["db"].(map[string]any)["host"].
func lookupMap(data map[string]any, key string) (any, bool) {
	parts := strings.Split(key, ".")
	current := any(data)

	for _, part := range parts {
		m, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		current, ok = m[part]
		if !ok {
			return nil, false
		}
	}

	return current, true
}

// setFieldFromAny sets a reflect.Value from a decoded file value (any type).
// JSON/YAML/TOML decoders produce native Go types: string, float64, bool,
// []any, map[string]any, etc.
func setFieldFromAny(field reflect.Value, val any) error {
	typ := field.Type()

	// Handle time.Duration specially.
	if typ == reflect.TypeFor[time.Duration]() {
		s, ok := val.(string)
		if !ok {
			return fmt.Errorf("expected string for time.Duration, got %T", val)
		}
		d, err := time.ParseDuration(s)
		if err != nil {
			return fmt.Errorf("cannot parse %q as time.Duration: %w", s, err)
		}
		field.Set(reflect.ValueOf(d))
		return nil
	}

	switch typ.Kind() {
	case reflect.String:
		s, ok := val.(string)
		if !ok {
			return fmt.Errorf("expected string, got %T", val)
		}
		field.SetString(s)

	case reflect.Int, reflect.Int64:
		switch v := val.(type) {
		case float64:
			field.SetInt(int64(v))
		case int:
			field.SetInt(int64(v))
		case int64:
			field.SetInt(v)
		default:
			return fmt.Errorf("expected number for %s, got %T", typ.Kind(), val)
		}

	case reflect.Float64:
		switch v := val.(type) {
		case float64:
			field.SetFloat(v)
		case int:
			field.SetFloat(float64(v))
		default:
			return fmt.Errorf("expected number for float64, got %T", val)
		}

	case reflect.Bool:
		b, ok := val.(bool)
		if !ok {
			return fmt.Errorf("expected bool, got %T", val)
		}
		field.SetBool(b)

	case reflect.Slice:
		return setSliceFromAny(field, val, typ)

	case reflect.Map:
		return setMapFromAny(field, val, typ)

	default:
		return fmt.Errorf("unsupported type %s", typ.Kind())
	}

	return nil
}

// setSliceFromAny converts a []any from a file decoder into a typed slice.
func setSliceFromAny(field reflect.Value, val any, typ reflect.Type) error {
	arr, ok := val.([]any)
	if !ok {
		return fmt.Errorf("expected array, got %T", val)
	}

	elemKind := typ.Elem().Kind()
	switch elemKind {
	case reflect.String:
		slice := make([]string, len(arr))
		for i, elem := range arr {
			s, ok := elem.(string)
			if !ok {
				return fmt.Errorf("expected string in array element %d, got %T", i, elem)
			}
			slice[i] = s
		}
		field.Set(reflect.ValueOf(slice))

	case reflect.Int:
		slice := make([]int, len(arr))
		for i, elem := range arr {
			switch v := elem.(type) {
			case float64:
				slice[i] = int(v)
			case int:
				slice[i] = v
			case int64:
				slice[i] = int(v)
			default:
				return fmt.Errorf("expected number in array element %d, got %T", i, elem)
			}
		}
		field.Set(reflect.ValueOf(slice))

	case reflect.Float64:
		slice := make([]float64, len(arr))
		for i, elem := range arr {
			switch v := elem.(type) {
			case float64:
				slice[i] = v
			case int:
				slice[i] = float64(v)
			case int64:
				slice[i] = float64(v)
			default:
				return fmt.Errorf("expected number in array element %d, got %T", i, elem)
			}
		}
		field.Set(reflect.ValueOf(slice))

	default:
		return fmt.Errorf("unsupported slice element type %s", elemKind)
	}

	return nil
}

// setMapFromAny converts a map[string]any from a file decoder into a typed map.
// Supports map[string]string and map[string]any.
func setMapFromAny(field reflect.Value, val any, typ reflect.Type) error {
	m, ok := val.(map[string]any)
	if !ok {
		return fmt.Errorf("expected map, got %T", val)
	}

	if typ.Key().Kind() != reflect.String {
		return fmt.Errorf("unsupported map key type %s", typ.Key().Kind())
	}

	elemKind := typ.Elem().Kind()
	switch {
	case elemKind == reflect.String:
		result := make(map[string]string, len(m))
		for k, v := range m {
			s, ok := v.(string)
			if !ok {
				return fmt.Errorf("expected string for map value %q, got %T", k, v)
			}
			result[k] = s
		}
		field.Set(reflect.ValueOf(result))

	case typ.Elem() == reflect.TypeFor[any]():
		result := make(map[string]any, len(m))
		for k, v := range m {
			result[k] = v
		}
		field.Set(reflect.ValueOf(result))

	default:
		return fmt.Errorf("unsupported map value type %s", typ.Elem())
	}

	return nil
}
