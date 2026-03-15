package gonfig

import (
	"reflect"
	"strings"
	"unicode"
)

// fieldInfo holds metadata about a single struct field extracted via reflection.
type fieldInfo struct {
	Name          string       // Go field name (e.g., "Host")
	Path          string       // Dot-separated path (e.g., "DB.Host")
	Type          reflect.Type // Field type
	DefaultVal    string       // Value from `default` tag
	HasDefault    bool         // Whether `default` tag was present
	EnvName       string       // Env var name (explicit or auto-derived)
	FlagName      string       // Flag name (explicit or auto-derived)
	ConfigKey     string       // Config file key (explicit or auto-derived)
	Description   string       // From `description` tag
	ValidateRules string       // From `validate` tag
	Index         []int        // Reflect index path for nested field access
}

// extractFields recursively walks a struct value and returns field metadata.
func extractFields(v reflect.Value, prefix string, indexPrefix []int) []fieldInfo {
	t := v.Type()
	var fields []fieldInfo

	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		if !sf.IsExported() {
			continue
		}

		path := sf.Name
		if prefix != "" {
			path = prefix + "." + sf.Name
		}

		idx := make([]int, len(indexPrefix)+1)
		copy(idx, indexPrefix)
		idx[len(indexPrefix)] = i

		// Recurse into nested structs (but not special types like time.Duration).
		if sf.Type.Kind() == reflect.Struct && sf.Type.PkgPath() != "time" {
			fields = append(fields, extractFields(v.Field(i), path, idx)...)
			continue
		}

		fi := fieldInfo{
			Name:  sf.Name,
			Path:  path,
			Type:  sf.Type,
			Index: idx,
		}

		if val, ok := sf.Tag.Lookup("default"); ok {
			fi.DefaultVal = val
			fi.HasDefault = true
		}
		fi.Description = sf.Tag.Get("description")
		fi.ValidateRules = sf.Tag.Get("validate")

		// Env name: explicit tag or auto-derived.
		if envTag := sf.Tag.Get("env"); envTag != "" {
			fi.EnvName = envTag
		} else {
			fi.EnvName = toEnvName(path)
		}

		// Flag name: explicit tag or auto-derived.
		if flagTag := sf.Tag.Get("flag"); flagTag != "" {
			fi.FlagName = flagTag
		} else {
			fi.FlagName = toFlagName(path)
		}

		// Config key: explicit tag or auto-derived.
		if keyTag := sf.Tag.Get("gonfig"); keyTag != "" {
			fi.ConfigKey = keyTag
		} else {
			fi.ConfigKey = toConfigKey(path)
		}

		fields = append(fields, fi)
	}

	return fields
}

// toEnvName converts a field path like "DB.Host" or "LogLevel" to "DB_HOST" or "LOG_LEVEL".
func toEnvName(path string) string {
	parts := strings.Split(path, ".")
	var envParts []string
	for _, p := range parts {
		envParts = append(envParts, camelToSnake(p))
	}
	return strings.ToUpper(strings.Join(envParts, "_"))
}

// toFlagName converts a field path like "DB.Host" or "LogLevel" to "db-host" or "log-level".
func toFlagName(path string) string {
	parts := strings.Split(path, ".")
	var flagParts []string
	for _, p := range parts {
		flagParts = append(flagParts, camelToSnake(p))
	}
	return strings.ToLower(strings.ReplaceAll(strings.Join(flagParts, "-"), "_", "-"))
}

// toConfigKey converts a field path like "DB.Host" or "LogLevel" to "db.host" or "log_level".
func toConfigKey(path string) string {
	parts := strings.Split(path, ".")
	var keyParts []string
	for _, p := range parts {
		keyParts = append(keyParts, strings.ToLower(camelToSnake(p)))
	}
	return strings.Join(keyParts, ".")
}

// camelToSnake converts a CamelCase string to snake_case.
// It handles consecutive uppercase letters (acronyms) by keeping them grouped:
// "MaxConn" → "max_conn", "DBHost" → "db_host", "HTTPSPort" → "https_port".
func camelToSnake(s string) string {
	var result strings.Builder
	runes := []rune(s)
	for i, r := range runes {
		if unicode.IsUpper(r) {
			if i > 0 {
				// Insert underscore before uppercase letter if:
				// - previous char is lowercase, OR
				// - previous char is uppercase AND next char is lowercase (end of acronym)
				prev := runes[i-1]
				if unicode.IsLower(prev) {
					result.WriteRune('_')
				} else if unicode.IsUpper(prev) && i+1 < len(runes) && unicode.IsLower(runes[i+1]) {
					result.WriteRune('_')
				}
			}
			result.WriteRune(r)
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}
