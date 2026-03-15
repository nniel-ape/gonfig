package gonfig

import (
	"flag"
	"fmt"
	"io"
	"reflect"
)

// applyFlags parses command-line flags from args and applies only explicitly-set
// flags to the target struct fields. Flags that are not provided in args do not
// modify the struct, preserving values set by earlier sources (file, env).
func applyFlags(target any, fields []fieldInfo, args []string) error {
	fs := flag.NewFlagSet("gonfig", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	// Register a string flag for each field. We store raw string values and
	// convert them later, which keeps flag registration simple and reuses
	// the existing setFieldValue type conversion logic.
	flagVals := make(map[string]*string, len(fields))
	shortToLong := make(map[string]string) // short flag name → long flag name
	for _, fi := range fields {
		if _, exists := flagVals[fi.FlagName]; exists {
			return fmt.Errorf("duplicate flag name %q for field %s", fi.FlagName, fi.Path)
		}
		val := ""
		flagVals[fi.FlagName] = &val
		defaultVal := fi.DefaultVal
		fs.StringVar(&val, fi.FlagName, defaultVal, fi.Description)

		// Register short flag alias pointing to the same value.
		if fi.ShortFlag != "" {
			if existingLong, exists := shortToLong[fi.ShortFlag]; exists {
				return fmt.Errorf("duplicate short flag %q for field %s (already used by %s)", fi.ShortFlag, fi.Path, existingLong)
			}
			shortToLong[fi.ShortFlag] = fi.FlagName
			fs.StringVar(&val, fi.ShortFlag, defaultVal, fi.Description)
		}
	}

	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("flag parsing: %w", err)
	}

	// Collect which flags were explicitly set on the command line.
	setFlags := make(map[string]bool)
	fs.Visit(func(f *flag.Flag) {
		setFlags[f.Name] = true
	})

	// Apply only explicitly-set flags to the struct.
	v := reflect.ValueOf(target).Elem()
	for _, fi := range fields {
		// Check if either the long flag or the short flag was explicitly set.
		if !setFlags[fi.FlagName] && !setFlags[fi.ShortFlag] {
			continue
		}

		raw := *flagVals[fi.FlagName]
		field := fieldByIndex(v, fi.Index)
		if !field.CanSet() {
			return fmt.Errorf("cannot set field %s", fi.Path)
		}

		if err := setFieldValue(field, raw); err != nil {
			return fmt.Errorf("flag --%s for %s: %w", fi.FlagName, fi.Path, err)
		}
	}

	return nil
}
