package gonfig

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
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
	var format string
	switch ext {
	case ".json":
		format = "json"
	case ".yaml", ".yml":
		format = "yaml"
	case ".toml":
		format = "toml"
	default:
		return fmt.Errorf("unsupported config file format: %s", ext)
	}

	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open config file: %w", err)
	}
	defer func() { _ = f.Close() }()

	data, err := decodeByFormat(f, format)
	if err != nil {
		return fmt.Errorf("decode %s: %w", ext, err)
	}

	return applyMap(target, data, fields)
}

// decodeByFormat decodes config data from a reader using the specified format.
func decodeByFormat(r io.Reader, format string) (map[string]any, error) {
	switch format {
	case "json":
		return decodeJSON(r)
	case "yaml":
		return decodeYAML(r)
	case "toml":
		return decodeTOML(r)
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
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

	for i := range fields {
		fi := &fields[i]
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
		return setIntFromAny(field, val, typ)

	case reflect.Float64:
		return setFloatFromAny(field, val)

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

// setIntFromAny sets an int or int64 field from a decoded file value.
func setIntFromAny(field reflect.Value, val any, typ reflect.Type) error {
	n, err := anyToInt64(val)
	if err != nil {
		return fmt.Errorf("expected number for %s, got %T", typ.Kind(), val)
	}
	if typ.Kind() == reflect.Int && (n > int64(math.MaxInt) || n < int64(math.MinInt)) {
		return fmt.Errorf("cannot convert %v to int: value out of range", val)
	}
	field.SetInt(n)
	return nil
}

// anyToInt64 converts a decoded file value (float64, int, int64) to int64.
func anyToInt64(val any) (int64, error) {
	switch v := val.(type) {
	case float64:
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return 0, fmt.Errorf("cannot convert %v to integer: value is not finite", v)
		}
		if v != math.Trunc(v) {
			return 0, fmt.Errorf("cannot convert %v to integer: value is not integral", v)
		}
		if v >= 1<<63 || v < -(1<<63) {
			return 0, fmt.Errorf("cannot convert %v to integer: value out of range", v)
		}
		return int64(v), nil
	case int:
		return int64(v), nil
	case int64:
		return v, nil
	default:
		return 0, fmt.Errorf("expected number, got %T", val)
	}
}

// setFloatFromAny sets a float64 field from a decoded file value.
func setFloatFromAny(field reflect.Value, val any) error {
	switch v := val.(type) {
	case float64:
		field.SetFloat(v)
	case int:
		field.SetFloat(float64(v))
	case int64:
		field.SetFloat(float64(v))
	default:
		return fmt.Errorf("expected number for float64, got %T", val)
	}
	return nil
}

// setSliceFromAny converts a []any from a file decoder into a typed slice.
// Each element is delegated to setFieldFromAny, so any scalar type supported
// by setFieldFromAny automatically works as a slice element.
func setSliceFromAny(field reflect.Value, val any, typ reflect.Type) error {
	arr, ok := val.([]any)
	if !ok {
		return fmt.Errorf("expected array, got %T", val)
	}

	slice := reflect.MakeSlice(typ, len(arr), len(arr))

	for i, elem := range arr {
		if err := setFieldFromAny(slice.Index(i), elem); err != nil {
			return fmt.Errorf("array element %d: %w", i, err)
		}
	}

	field.Set(slice)
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
		field.Set(reflect.ValueOf(m))

	default:
		return fmt.Errorf("unsupported map value type %s", typ.Elem())
	}

	return nil
}
