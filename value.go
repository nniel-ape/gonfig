package gonfig

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// setFieldValue sets a reflect.Value from a raw string, performing type conversion.
// Supported types: string, int, int64, float64, bool, time.Duration, []string, []int.
func setFieldValue(field reflect.Value, raw string) error {
	typ := field.Type()

	// Handle time.Duration specially since it's a named int64.
	if typ == reflect.TypeFor[time.Duration]() {
		d, err := time.ParseDuration(raw)
		if err != nil {
			return fmt.Errorf("cannot parse %q as time.Duration: %w", raw, err)
		}
		field.Set(reflect.ValueOf(d))
		return nil
	}

	switch typ.Kind() {
	case reflect.String:
		field.SetString(raw)

	case reflect.Int, reflect.Int64:
		v, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return fmt.Errorf("cannot parse %q as %s: %w", raw, typ.Kind(), err)
		}
		field.SetInt(v)

	case reflect.Float64:
		v, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return fmt.Errorf("cannot parse %q as float64: %w", raw, err)
		}
		field.SetFloat(v)

	case reflect.Bool:
		v, err := strconv.ParseBool(raw)
		if err != nil {
			return fmt.Errorf("cannot parse %q as bool: %w", raw, err)
		}
		field.SetBool(v)

	case reflect.Slice:
		return setSliceValue(field, raw, typ)

	default:
		return fmt.Errorf("unsupported type %s", typ.Kind())
	}

	return nil
}

// setSliceValue parses a comma-separated string into a slice field.
func setSliceValue(field reflect.Value, raw string, typ reflect.Type) error {
	if raw == "" {
		field.Set(reflect.MakeSlice(typ, 0, 0))
		return nil
	}

	parts := strings.Split(raw, ",")
	elemKind := typ.Elem().Kind()

	switch elemKind {
	case reflect.String:
		slice := make([]string, len(parts))
		for i, p := range parts {
			slice[i] = strings.TrimSpace(p)
		}
		field.Set(reflect.ValueOf(slice))

	case reflect.Int:
		slice := make([]int, len(parts))
		for i, p := range parts {
			v, err := strconv.Atoi(strings.TrimSpace(p))
			if err != nil {
				return fmt.Errorf("cannot parse %q as int in slice element %d: %w", p, i, err)
			}
			slice[i] = v
		}
		field.Set(reflect.ValueOf(slice))

	default:
		return fmt.Errorf("unsupported slice element type %s", elemKind)
	}

	return nil
}

// fieldByIndex returns the nested field of a struct value at the given index path.
// It ensures intermediate struct fields are valid and settable.
func fieldByIndex(v reflect.Value, index []int) reflect.Value {
	for _, i := range index {
		v = v.Field(i)
	}
	return v
}
