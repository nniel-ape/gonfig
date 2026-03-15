package gonfig

import (
	"fmt"
	"reflect"
)

// applyDefaults sets struct fields from their `default` tag values.
// Fields without a `default` tag are left unchanged.
func applyDefaults(target any, fields []fieldInfo) error {
	v := reflect.ValueOf(target).Elem()

	for i := range fields {
		fi := &fields[i]
		if !fi.HasDefault {
			continue
		}

		field := fieldByIndex(v, fi.Index)
		if !field.CanSet() {
			return fmt.Errorf("cannot set field %s", fi.Path)
		}

		if err := setFieldValue(field, fi.DefaultVal); err != nil {
			return fmt.Errorf("default for %s: %w", fi.Path, err)
		}
	}

	return nil
}
