package gonfig

import (
	"reflect"
	"strings"
	"unicode"
)

// timePkgPath is the package path for the time package, used to detect time.Duration.
const timePkgPath = "time"

// fieldInfo holds metadata about a single struct field extracted via reflection.
type fieldInfo struct {
	Name          string       // Go field name (e.g., "Host")
	Path          string       // Dot-separated path (e.g., "DB.Host")
	Type          reflect.Type // Field type
	DefaultVal    string       // Value from `default` tag
	HasDefault    bool         // Whether `default` tag was present
	EnvName       string       // Env var name (explicit or auto-derived)
	FlagName      string       // Flag name (explicit or auto-derived)
	ShortFlag     string       // Short flag name (explicit only, e.g. "p" for -p)
	ConfigKey     string       // Config file key (explicit or auto-derived)
	Description   string       // From `description` tag
	ValidateRules string       // From `validate` tag
	Index         []int        // Reflect index path for nested field access
}

// extractFields recursively walks a struct value and returns field metadata.
func extractFields(v reflect.Value, prefix string, indexPrefix []int) []fieldInfo {
	t := v.Type()

	var fields []fieldInfo

	for i := range t.NumField() {
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
		// The gonfig tag on a struct field overrides the path segment for all children.
		if sf.Type.Kind() == reflect.Struct && sf.Type.PkgPath() != timePkgPath {
			if keyTag := sf.Tag.Get("gonfig"); keyTag != "" {
				path = keyTag
				if prefix != "" {
					path = prefix + "." + keyTag
				}
			}

			fields = append(fields, extractFields(v.Field(i), path, idx)...)

			continue
		}

		fields = append(fields, buildLeafField(&sf, path, idx))
	}

	return fields
}

// buildLeafField creates a fieldInfo for a non-struct field, reading all tags.
func buildLeafField(sf *reflect.StructField, path string, idx []int) fieldInfo {
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
	fi.ShortFlag = sf.Tag.Get("short")

	if envTag := sf.Tag.Get("env"); envTag != "" {
		fi.EnvName = envTag
	} else {
		fi.EnvName = toEnvName(path)
	}

	if flagTag := sf.Tag.Get("flag"); flagTag != "" {
		fi.FlagName = flagTag
	} else {
		fi.FlagName = toFlagName(path)
	}

	if keyTag := sf.Tag.Get("gonfig"); keyTag != "" {
		fi.ConfigKey = keyTag
	} else {
		fi.ConfigKey = toConfigKey(path)
	}

	return fi
}

// toEnvName converts a field path like "DB.Host" or "LogLevel" to "DB_HOST" or "LOG_LEVEL".
func toEnvName(path string) string {
	parts := strings.Split(path, ".")

	envParts := make([]string, 0, len(parts))
	for _, p := range parts {
		envParts = append(envParts, camelToSnake(p))
	}

	return strings.ToUpper(strings.Join(envParts, "_"))
}

// toFlagName converts a field path like "DB.Host" or "LogLevel" to "db-host" or "log-level".
func toFlagName(path string) string {
	parts := strings.Split(path, ".")

	flagParts := make([]string, 0, len(parts))
	for _, p := range parts {
		flagParts = append(flagParts, camelToSnake(p))
	}

	return strings.ToLower(strings.ReplaceAll(strings.Join(flagParts, "-"), "_", "-"))
}

// toConfigKey converts a field path like "DB.Host" or "LogLevel" to "db.host" or "log_level".
func toConfigKey(path string) string {
	parts := strings.Split(path, ".")

	keyParts := make([]string, 0, len(parts))
	for _, p := range parts {
		keyParts = append(keyParts, strings.ToLower(camelToSnake(p)))
	}

	return strings.Join(keyParts, ".")
}

// knownAcronyms lists common acronyms recognized during CamelCase splitting.
// Sorted by length descending for greedy (longest-first) matching.
var knownAcronyms = []string{
	"HTTPS", "HTTP",
	"URL", "URI", "API", "SQL", "DNS", "SSH", "SSL", "TLS", "TCP", "UDP", "RPC",
	"ID", "IP",
}

// acronymMatchAt returns the length of a known acronym at position pos in runes,
// or 0 if no acronym matches with a valid word boundary.
func acronymMatchAt(runes []rune, pos int) int {
	for _, acr := range knownAcronyms {
		acrRunes := []rune(acr)

		n := len(acrRunes)
		if pos+n > len(runes) {
			continue
		}

		if string(runes[pos:pos+n]) != acr {
			continue
		}

		end := pos + n
		// End of string: always valid.
		if end >= len(runes) {
			return n
		}
		// Followed by non-uppercase (lowercase suffix like 's' in "IDs"): valid.
		if !unicode.IsUpper(runes[end]) {
			return n
		}
		// Followed by uppercase that starts a CamelCase word (upper then non-upper): valid.
		if end+1 < len(runes) && !unicode.IsUpper(runes[end+1]) {
			return n
		}
		// Single uppercase char at end of string: valid.
		if end+1 >= len(runes) {
			return n
		}
		// Followed by another known acronym: valid.
		if acronymMatchAt(runes, end) > 0 {
			return n
		}
		// Otherwise this would split a non-acronym uppercase sequence (e.g., "ID" in "IDEA").
	}

	return 0
}

// camelToSnake converts a CamelCase string to snake_case, recognizing known acronyms.
// It handles consecutive uppercase letters by splitting at acronym boundaries:
// "APIURL" → "API_URL", "MarketIDs" → "Market_IDs", "HTTPSPort" → "HTTPS_Port".
func camelToSnake(s string) string {
	if s == "" {
		return s
	}

	return strings.Join(splitCamelWords([]rune(s)), "_")
}

// splitCamelWords splits a CamelCase rune sequence into words, recognizing known acronyms.
func splitCamelWords(runes []rune) []string {
	var words []string

	i := 0

	for i < len(runes) {
		if unicode.IsUpper(runes[i]) {
			word, next := collectUpperWord(runes, i)
			words = append(words, word)
			i = next
		} else {
			word, next := collectNonUpperRun(runes, i)
			words = append(words, word)
			i = next
		}
	}

	return words
}

// collectUpperWord collects one word starting with an uppercase letter.
// It tries an acronym match first, then falls back to standard CamelCase.
func collectUpperWord(runes []rune, i int) (result string, next int) {
	// Try acronym match first.
	if n := acronymMatchAt(runes, i); n > 0 {
		end := i + n
		// Include trailing non-uppercase suffix (e.g., 's' in "IDs").
		for end < len(runes) && !unicode.IsUpper(runes[end]) {
			end++
		}

		return string(runes[i:end]), end
	}

	// Standard CamelCase word.
	var word strings.Builder
	word.WriteRune(runes[i])
	i++
	// Collect consecutive uppercase that aren't word boundaries.
	for i < len(runes) && unicode.IsUpper(runes[i]) {
		if i+1 < len(runes) && !unicode.IsUpper(runes[i+1]) {
			break // Next char starts a CamelCase word.
		}

		if acronymMatchAt(runes, i) > 0 {
			break // A known acronym starts here.
		}

		word.WriteRune(runes[i])
		i++
	}
	// Collect trailing non-uppercase chars.
	for i < len(runes) && !unicode.IsUpper(runes[i]) {
		word.WriteRune(runes[i])
		i++
	}

	return word.String(), i
}

// collectNonUpperRun collects consecutive non-uppercase characters as a single word.
func collectNonUpperRun(runes []rune, i int) (result string, next int) {
	start := i
	for i < len(runes) && !unicode.IsUpper(runes[i]) {
		i++
	}

	return string(runes[start:i]), i
}
