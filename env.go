package gonfig

import (
	"fmt"
	"os"
	"reflect"
)

// applyEnv sets struct fields from environment variables.
// For each field, it looks up the env var name (with optional prefix prepended)
// using os.LookupEnv. If the env var is set, its value is parsed and applied.
// Fields whose env vars are not set are left unchanged.
func applyEnv(target any, fields []fieldInfo, prefix string) error {
	v := reflect.ValueOf(target).Elem()

	for i := range fields {
		fi := &fields[i]
		envName := fi.EnvName
		if prefix != "" {
			envName = prefix + "_" + fi.EnvName
		}

		raw, ok := os.LookupEnv(envName)
		if !ok {
			continue
		}

		field := fieldByIndex(v, fi.Index)
		if !field.CanSet() {
			return fmt.Errorf("cannot set field %s", fi.Path)
		}

		if err := setFieldValue(field, raw); err != nil {
			return fmt.Errorf("env %s for %s: %w", envName, fi.Path, err)
		}
	}

	return nil
}
