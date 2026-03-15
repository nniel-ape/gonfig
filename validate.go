package gonfig

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// FieldError describes a validation failure for a single struct field.
type FieldError struct {
	Field   string // Dot-separated field path (e.g., "DB.Port")
	Rule    string // The validation rule that failed (e.g., "required", "min=1")
	Message string // Human-readable error message
}

func (e FieldError) Error() string {
	return fmt.Sprintf("field %s: %s", e.Field, e.Message)
}

// ValidationError collects all field validation errors.
// It implements the error interface and wraps ErrValidation for errors.Is support.
type ValidationError struct {
	Errors []FieldError
}

func (e *ValidationError) Error() string {
	msgs := make([]string, len(e.Errors))
	for i, fe := range e.Errors {
		msgs[i] = fe.Error()
	}
	return fmt.Sprintf("validation failed: %s", strings.Join(msgs, "; "))
}

func (e *ValidationError) Unwrap() error {
	return ErrValidation
}

// validate checks all fields with validate tags and returns a ValidationError
// containing all failures. It does not fail on the first error.
func validate(target any, fields []fieldInfo) error {
	rv := reflect.ValueOf(target).Elem()
	var errs []FieldError

	for _, fi := range fields {
		if fi.ValidateRules == "" {
			continue
		}

		fv := fieldByIndex(rv, fi.Index)
		rules := strings.Split(fi.ValidateRules, ",")

		for _, rule := range rules {
			rule = strings.TrimSpace(rule)
			if rule == "" {
				continue
			}

			if fe, ok := checkRule(fi, fv, rule); !ok {
				errs = append(errs, fe)
			}
		}
	}

	if len(errs) > 0 {
		return &ValidationError{Errors: errs}
	}
	return nil
}

// checkRule evaluates a single validation rule against a field value.
// Returns the FieldError and false if validation fails.
func checkRule(fi fieldInfo, fv reflect.Value, rule string) (FieldError, bool) {
	switch {
	case rule == "required":
		return checkRequired(fi, fv)
	case strings.HasPrefix(rule, "min="):
		return checkMin(fi, fv, rule)
	case strings.HasPrefix(rule, "max="):
		return checkMax(fi, fv, rule)
	case strings.HasPrefix(rule, "oneof="):
		return checkOneof(fi, fv, rule)
	default:
		return FieldError{
			Field:   fi.Path,
			Rule:    rule,
			Message: fmt.Sprintf("unknown validation rule: %s", rule),
		}, false
	}
}

// checkRequired verifies the field is not the zero value for its type.
func checkRequired(fi fieldInfo, fv reflect.Value) (FieldError, bool) {
	if fv.IsZero() {
		return FieldError{
			Field:   fi.Path,
			Rule:    "required",
			Message: "required field is empty",
		}, false
	}
	return FieldError{}, true
}

// checkMin verifies numeric fields are >= the minimum value.
func checkMin(fi fieldInfo, fv reflect.Value, rule string) (FieldError, bool) {
	boundStr := strings.TrimPrefix(rule, "min=")
	bound, err := strconv.ParseFloat(boundStr, 64)
	if err != nil {
		return FieldError{
			Field:   fi.Path,
			Rule:    rule,
			Message: fmt.Sprintf("invalid min value: %s", boundStr),
		}, false
	}

	val, ok := numericValue(fv)
	if !ok {
		return FieldError{
			Field:   fi.Path,
			Rule:    rule,
			Message: fmt.Sprintf("min rule requires a numeric type, got %s", fi.Type.Kind()),
		}, false
	}

	if val < bound {
		return FieldError{
			Field:   fi.Path,
			Rule:    rule,
			Message: fmt.Sprintf("value %v is less than minimum %s", val, boundStr),
		}, false
	}
	return FieldError{}, true
}

// checkMax verifies numeric fields are <= the maximum value.
func checkMax(fi fieldInfo, fv reflect.Value, rule string) (FieldError, bool) {
	boundStr := strings.TrimPrefix(rule, "max=")
	bound, err := strconv.ParseFloat(boundStr, 64)
	if err != nil {
		return FieldError{
			Field:   fi.Path,
			Rule:    rule,
			Message: fmt.Sprintf("invalid max value: %s", boundStr),
		}, false
	}

	val, ok := numericValue(fv)
	if !ok {
		return FieldError{
			Field:   fi.Path,
			Rule:    rule,
			Message: fmt.Sprintf("max rule requires a numeric type, got %s", fi.Type.Kind()),
		}, false
	}

	if val > bound {
		return FieldError{
			Field:   fi.Path,
			Rule:    rule,
			Message: fmt.Sprintf("value %v is greater than maximum %s", val, boundStr),
		}, false
	}
	return FieldError{}, true
}

// checkOneof verifies the field value is one of the allowed values.
func checkOneof(fi fieldInfo, fv reflect.Value, rule string) (FieldError, bool) {
	allowedStr := strings.TrimPrefix(rule, "oneof=")
	allowed := strings.Split(allowedStr, " ")

	val := fmt.Sprintf("%v", fv.Interface())
	for _, a := range allowed {
		if val == a {
			return FieldError{}, true
		}
	}

	return FieldError{
		Field:   fi.Path,
		Rule:    rule,
		Message: fmt.Sprintf("value %q is not one of [%s]", val, strings.Join(allowed, ", ")),
	}, false
}

// numericValue extracts the float64 representation of a numeric reflect.Value.
func numericValue(fv reflect.Value) (float64, bool) {
	switch fv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(fv.Int()), true
	case reflect.Float32, reflect.Float64:
		return fv.Float(), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(fv.Uint()), true
	default:
		return 0, false
	}
}
