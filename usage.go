package gonfig

import (
	"fmt"
	"reflect"
	"strings"
)

// Usage generates formatted usage text from the struct metadata of the target.
// The target must be a pointer to a struct (same as Load).
// Options like WithEnvPrefix affect the displayed env var names.
//
// The output groups fields by nested struct sections and displays each field's
// flag name, env var name, type, default value, and description in aligned columns.
func Usage(target any, opts ...Option) string {
	if target == nil {
		return ""
	}

	rv := reflect.ValueOf(target)
	if rv.Kind() != reflect.Pointer || rv.Elem().Kind() != reflect.Struct {
		return ""
	}

	var o options
	for _, opt := range opts {
		opt(&o)
	}

	fields := extractFields(rv.Elem(), "", nil)

	// Build section groups from the struct hierarchy.
	sections := buildSections(rv.Elem().Type(), fields, o.envPrefix)

	var b strings.Builder
	for i, sec := range sections {
		if i > 0 {
			b.WriteString("\n")
		}
		if sec.name != "" {
			b.WriteString(sec.name)
			b.WriteString(":\n")
		}
		writeSection(&b, sec.entries)
	}

	return b.String()
}

// section represents a group of fields, optionally under a named header.
type section struct {
	name    string
	entries []usageEntry
}

// usageEntry holds the formatted parts of a single field's usage line.
type usageEntry struct {
	flag        string
	env         string
	typeName    string
	defaultVal  string
	description string
}

// buildSections groups fields into sections based on struct nesting.
func buildSections(t reflect.Type, fields []fieldInfo, envPrefix string) []section {
	// Determine which top-level struct fields are nested structs (sections).
	var sectionOrder []string
	sectionSet := make(map[string]bool)

	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		if !sf.IsExported() {
			continue
		}
		if sf.Type.Kind() == reflect.Struct && sf.Type.PkgPath() != "time" {
			sectionOrder = append(sectionOrder, sf.Name)
			sectionSet[sf.Name] = true
		}
	}

	// Partition fields into root and nested sections.
	rootEntries := []usageEntry{}
	sectionEntries := make(map[string][]usageEntry)

	for _, fi := range fields {
		entry := fieldToEntry(fi, envPrefix)

		// Check if this field belongs to a section.
		parts := strings.SplitN(fi.Path, ".", 2)
		if len(parts) > 1 && sectionSet[parts[0]] {
			sectionEntries[parts[0]] = append(sectionEntries[parts[0]], entry)
		} else {
			rootEntries = append(rootEntries, entry)
		}
	}

	var sections []section

	// Root fields first (no header).
	if len(rootEntries) > 0 {
		sections = append(sections, section{name: "", entries: rootEntries})
	}

	// Then each nested struct section.
	for _, name := range sectionOrder {
		if entries, ok := sectionEntries[name]; ok && len(entries) > 0 {
			sections = append(sections, section{name: name, entries: entries})
		}
	}

	return sections
}

// fieldToEntry converts a fieldInfo into a usageEntry.
func fieldToEntry(fi fieldInfo, envPrefix string) usageEntry {
	envName := fi.EnvName
	if envPrefix != "" {
		envName = envPrefix + "_" + envName
	}

	return usageEntry{
		flag:        "--" + fi.FlagName,
		env:         envName,
		typeName:    friendlyTypeName(fi.Type),
		defaultVal:  fi.DefaultVal,
		description: fi.Description,
	}
}

// friendlyTypeName returns a human-readable type name for usage display.
func friendlyTypeName(t reflect.Type) string {
	switch t.Kind() {
	case reflect.String:
		return "string"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32:
		return "int"
	case reflect.Int64:
		if t.PkgPath() == "time" && t.Name() == "Duration" {
			return "duration"
		}
		return "int"
	case reflect.Float32, reflect.Float64:
		return "float"
	case reflect.Bool:
		return "bool"
	case reflect.Slice:
		return "[]" + friendlyTypeName(t.Elem())
	case reflect.Map:
		return "map[" + friendlyTypeName(t.Key()) + "]" + friendlyTypeName(t.Elem())
	default:
		return t.String()
	}
}

// writeSection writes aligned usage entries to the builder.
func writeSection(b *strings.Builder, entries []usageEntry) {
	if len(entries) == 0 {
		return
	}

	// Calculate column widths.
	var maxFlag, maxEnv, maxType, maxDefault int
	for _, e := range entries {
		if len(e.flag) > maxFlag {
			maxFlag = len(e.flag)
		}
		if len(e.env) > maxEnv {
			maxEnv = len(e.env)
		}
		if len(e.typeName) > maxType {
			maxType = len(e.typeName)
		}
		defStr := formatDefault(e.defaultVal)
		if len(defStr) > maxDefault {
			maxDefault = len(defStr)
		}
	}

	for _, e := range entries {
		defStr := formatDefault(e.defaultVal)
		line := fmt.Sprintf("  %-*s  %-*s  %-*s  %-*s",
			maxFlag, e.flag,
			maxEnv, e.env,
			maxType, e.typeName,
			maxDefault, defStr,
		)
		if e.description != "" {
			line += "  " + e.description
		}
		b.WriteString(strings.TrimRight(line, " "))
		b.WriteString("\n")
	}
}

// formatDefault formats a default value for display.
func formatDefault(val string) string {
	if val == "" {
		return ""
	}
	return "(default: " + val + ")"
}
