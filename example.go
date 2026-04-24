package gonfig

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// Example generates a config file skeleton from the struct metadata of target.
// The target must be a pointer to a struct (same as Load).
// Options like WithEnvPrefix are accepted for consistency but do not affect the output.
//
// For YAML and TOML formats, comments include each field's description and validation rules.
// For JSON, a plain skeleton is produced (JSON has no comment syntax).
// Fields with defaults show their default value; fields without defaults show zero values.
func Example(target any, format Format, opts ...Option) string {
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
	root := buildExampleTree(fields)

	switch format {
	case YAML:
		return renderYAML(root)
	case JSON:
		return renderJSON(root)
	case TOML:
		return renderTOML(root)
	default:
		return renderYAML(root)
	}
}

// exampleNode represents a leaf or branch in the config tree.
type exampleNode struct {
	key      string
	value    any
	comment  string
	children []*exampleNode
}

// buildExampleTree converts flat fieldInfo list into a tree of exampleNode,
// reconstructing nesting from dot-separated ConfigKey paths.
func buildExampleTree(fields []fieldInfo) *exampleNode {
	root := &exampleNode{}

	for i := range fields {
		fi := &fields[i]
		segments := strings.Split(fi.ConfigKey, ".")
		current := root

		for j, seg := range segments {
			if j == len(segments)-1 {
				current.children = append(current.children, &exampleNode{
					key:     seg,
					value:   fieldValueForExample(fi),
					comment: buildComment(fi),
				})
			} else {
				found := false
				for _, child := range current.children {
					if child.key == seg {
						current = child
						found = true
						break
					}
				}
				if !found {
					branch := &exampleNode{key: seg}
					current.children = append(current.children, branch)
					current = branch
				}
			}
		}
	}

	return root
}

// fieldValueForExample returns a Go value suitable for rendering in the config skeleton.
func fieldValueForExample(fi *fieldInfo) any {
	typ := fi.Type

	if typ.Kind() == reflect.Map {
		return reflect.MakeMap(typ).Interface()
	}

	v := reflect.New(typ).Elem()
	if fi.HasDefault {
		if err := setFieldValue(v, fi.DefaultVal); err != nil {
			// Fall through to zero value on parse error.
		}
	}

	if typ == reflect.TypeFor[time.Duration]() {
		return v.Interface().(time.Duration).String()
	}

	return v.Interface()
}

// buildComment combines description and validation rules into a comment string.
func buildComment(fi *fieldInfo) string {
	var parts []string
	if fi.Description != "" {
		parts = append(parts, fi.Description)
	}
	if fi.ValidateRules != "" {
		parts = append(parts, "("+fi.ValidateRules+")")
	}
	return strings.Join(parts, " ")
}

// renderYAML produces a YAML config skeleton with comments.
func renderYAML(root *exampleNode) string {
	var b strings.Builder
	for i, child := range root.children {
		if i > 0 {
			b.WriteByte('\n')
		}
		renderYAMLNode(&b, child, 0)
	}
	return b.String()
}

func renderYAMLNode(b *strings.Builder, node *exampleNode, indent int) {
	prefix := strings.Repeat("  ", indent)

	if len(node.children) > 0 {
		if node.comment != "" {
			fmt.Fprintf(b, "%s# %s\n", prefix, node.comment)
		}
		fmt.Fprintf(b, "%s%s:\n", prefix, node.key)
		for i, child := range node.children {
			if i > 0 {
				b.WriteByte('\n')
			}
			renderYAMLNode(b, child, indent+1)
		}
		return
	}

	if node.comment != "" {
		fmt.Fprintf(b, "%s# %s\n", prefix, node.comment)
	}
	fmt.Fprintf(b, "%s%s: %s\n", prefix, node.key, formatYAMLValue(node.value))
}

func formatYAMLValue(v any) string {
	switch val := v.(type) {
	case string:
		return strconv.Quote(val)
	case int:
		return strconv.Itoa(val)
	case int64:
		return strconv.FormatInt(val, 10)
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(val)
	case []string:
		return formatSliceYAML(val, strconv.Quote)
	case []int:
		return formatSliceYAML(val, strconv.Itoa)
	case []float64:
		return formatSliceYAML(val, func(f float64) string { return strconv.FormatFloat(f, 'f', -1, 64) })
	case []bool:
		return formatSliceYAML(val, strconv.FormatBool)
	default:
		rv := reflect.ValueOf(v)
		if rv.Kind() == reflect.Map {
			return "{}"
		}
		return fmt.Sprintf("%v", v)
	}
}

func formatSliceYAML[T any](s []T, fmtElem func(T) string) string {
	if len(s) == 0 {
		return "[]"
	}
	items := make([]string, len(s))
	for i, elem := range s {
		items[i] = fmtElem(elem)
	}
	return "[" + strings.Join(items, ", ") + "]"
}

// renderJSON produces a JSON config skeleton (no comments).
func renderJSON(root *exampleNode) string {
	m := exampleTreeToMap(root)
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return "{}\n"
	}
	return string(data) + "\n"
}

func exampleTreeToMap(node *exampleNode) map[string]any {
	m := make(map[string]any, len(node.children))
	for _, child := range node.children {
		if len(child.children) > 0 {
			m[child.key] = exampleTreeToMap(child)
		} else {
			m[child.key] = child.value
		}
	}
	return m
}

// renderTOML produces a TOML config skeleton with comments.
func renderTOML(root *exampleNode) string {
	var b strings.Builder
	for i, child := range root.children {
		if i > 0 {
			b.WriteByte('\n')
		}
		renderTOMLNode(&b, child, "")
	}
	return b.String()
}

func renderTOMLNode(b *strings.Builder, node *exampleNode, path string) {
	if len(node.children) == 0 {
		// Leaf node (root-level or under a section).
		if node.comment != "" {
			fmt.Fprintf(b, "# %s\n", node.comment)
		}
		fmt.Fprintf(b, "%s = %s\n", node.key, formatTOMLValue(node.value))
		return
	}

	// Branch node.
	sectionPath := node.key
	if path != "" {
		sectionPath = path + "." + node.key
	}
	fmt.Fprintf(b, "[%s]\n", sectionPath)

	// Emit leaf children first.
	for _, child := range node.children {
		if len(child.children) == 0 {
			if child.comment != "" {
				fmt.Fprintf(b, "# %s\n", child.comment)
			}
			fmt.Fprintf(b, "%s = %s\n", child.key, formatTOMLValue(child.value))
		}
	}

	// Then emit sub-tables.
	for _, child := range node.children {
		if len(child.children) > 0 {
			b.WriteByte('\n')
			renderTOMLNode(b, child, sectionPath)
		}
	}
}

func formatTOMLValue(v any) string {
	switch val := v.(type) {
	case string:
		return strconv.Quote(val)
	case int:
		return strconv.Itoa(val)
	case int64:
		return strconv.FormatInt(val, 10)
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(val)
	case []string:
		return formatSliceTOML(val, strconv.Quote)
	case []int:
		return formatSliceTOML(val, strconv.Itoa)
	case []float64:
		return formatSliceTOML(val, func(f float64) string { return strconv.FormatFloat(f, 'f', -1, 64) })
	case []bool:
		return formatSliceTOML(val, strconv.FormatBool)
	default:
		rv := reflect.ValueOf(v)
		if rv.Kind() == reflect.Map {
			return "{}"
		}
		return fmt.Sprintf("%v", v)
	}
}

func formatSliceTOML[T any](s []T, fmtElem func(T) string) string {
	if len(s) == 0 {
		return "[]"
	}
	items := make([]string, len(s))
	for i, elem := range s {
		items[i] = fmtElem(elem)
	}
	return "[" + strings.Join(items, ", ") + "]"
}
