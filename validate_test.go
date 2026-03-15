package gonfig

import (
	"errors"
	"reflect"
	"testing"
)

func TestValidateRequired(t *testing.T) {
	tests := []struct {
		name    string
		target  any
		wantErr bool
	}{
		{
			name: "zero string fails",
			target: &struct {
				Name string `validate:"required"`
			}{Name: ""},
			wantErr: true,
		},
		{
			name: "non-zero string passes",
			target: &struct {
				Name string `validate:"required"`
			}{Name: "alice"},
			wantErr: false,
		},
		{
			name: "zero int fails",
			target: &struct {
				Port int `validate:"required"`
			}{Port: 0},
			wantErr: true,
		},
		{
			name: "non-zero int passes",
			target: &struct {
				Port int `validate:"required"`
			}{Port: 8080},
			wantErr: false,
		},
		{
			name: "zero bool fails",
			target: &struct {
				Debug bool `validate:"required"`
			}{Debug: false},
			wantErr: true,
		},
		{
			name: "true bool passes",
			target: &struct {
				Debug bool `validate:"required"`
			}{Debug: true},
			wantErr: false,
		},
		{
			name: "zero float fails",
			target: &struct {
				Rate float64 `validate:"required"`
			}{Rate: 0.0},
			wantErr: true,
		},
		{
			name: "non-zero float passes",
			target: &struct {
				Rate float64 `validate:"required"`
			}{Rate: 3.14},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rv := reflect.ValueOf(tt.target).Elem()
			fields := extractFields(rv, "", nil)
			err := validate(tt.target, fields)

			if tt.wantErr && err == nil {
				t.Fatalf("expected validation error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if tt.wantErr {
				var ve *ValidationError
				if !errors.As(err, &ve) {
					t.Fatalf("expected *ValidationError, got %T", err)
				}
				if len(ve.Errors) == 0 {
					t.Fatal("expected at least one FieldError")
				}
				if ve.Errors[0].Rule != "required" {
					t.Errorf("expected rule 'required', got %q", ve.Errors[0].Rule)
				}
			}
		})
	}
}

func TestValidateMinMax(t *testing.T) {
	tests := []struct {
		name    string
		target  any
		wantErr bool
		rule    string
	}{
		{
			name: "int above min passes",
			target: &struct {
				Port int `validate:"min=1"`
			}{Port: 80},
			wantErr: false,
		},
		{
			name: "int at min passes",
			target: &struct {
				Port int `validate:"min=1"`
			}{Port: 1},
			wantErr: false,
		},
		{
			name: "int below min fails",
			target: &struct {
				Port int `validate:"min=1"`
			}{Port: 0},
			wantErr: true,
			rule:    "min=1",
		},
		{
			name: "int below max passes",
			target: &struct {
				Port int `validate:"max=65535"`
			}{Port: 8080},
			wantErr: false,
		},
		{
			name: "int at max passes",
			target: &struct {
				Port int `validate:"max=65535"`
			}{Port: 65535},
			wantErr: false,
		},
		{
			name: "int above max fails",
			target: &struct {
				Port int `validate:"max=65535"`
			}{Port: 70000},
			wantErr: true,
			rule:    "max=65535",
		},
		{
			name: "float above min passes",
			target: &struct {
				Rate float64 `validate:"min=0.5"`
			}{Rate: 1.0},
			wantErr: false,
		},
		{
			name: "float below min fails",
			target: &struct {
				Rate float64 `validate:"min=0.5"`
			}{Rate: 0.1},
			wantErr: true,
			rule:    "min=0.5",
		},
		{
			name: "float above max fails",
			target: &struct {
				Rate float64 `validate:"max=100.5"`
			}{Rate: 200.0},
			wantErr: true,
			rule:    "max=100.5",
		},
		{
			name: "float at max passes",
			target: &struct {
				Rate float64 `validate:"max=100.5"`
			}{Rate: 100.5},
			wantErr: false,
		},
		{
			name: "negative int with min passes",
			target: &struct {
				Temp int `validate:"min=-50"`
			}{Temp: -10},
			wantErr: false,
		},
		{
			name: "negative int below min fails",
			target: &struct {
				Temp int `validate:"min=-50"`
			}{Temp: -100},
			wantErr: true,
			rule:    "min=-50",
		},
		{
			name: "min on string type fails with type error",
			target: &struct {
				Name string `validate:"min=1"`
			}{Name: "hello"},
			wantErr: true,
			rule:    "min=1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rv := reflect.ValueOf(tt.target).Elem()
			fields := extractFields(rv, "", nil)
			err := validate(tt.target, fields)

			if tt.wantErr && err == nil {
				t.Fatalf("expected validation error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if tt.wantErr {
				var ve *ValidationError
				if !errors.As(err, &ve) {
					t.Fatalf("expected *ValidationError, got %T", err)
				}
				if tt.rule != "" && ve.Errors[0].Rule != tt.rule {
					t.Errorf("expected rule %q, got %q", tt.rule, ve.Errors[0].Rule)
				}
			}
		})
	}
}

func TestValidateOneof(t *testing.T) {
	tests := []struct {
		name    string
		target  any
		wantErr bool
	}{
		{
			name: "valid value passes",
			target: &struct {
				Level string `validate:"oneof=debug info warn error"`
			}{Level: "info"},
			wantErr: false,
		},
		{
			name: "first option passes",
			target: &struct {
				Level string `validate:"oneof=debug info warn error"`
			}{Level: "debug"},
			wantErr: false,
		},
		{
			name: "last option passes",
			target: &struct {
				Level string `validate:"oneof=debug info warn error"`
			}{Level: "error"},
			wantErr: false,
		},
		{
			name: "invalid value fails",
			target: &struct {
				Level string `validate:"oneof=debug info warn error"`
			}{Level: "trace"},
			wantErr: true,
		},
		{
			name: "empty value fails",
			target: &struct {
				Level string `validate:"oneof=debug info warn error"`
			}{Level: ""},
			wantErr: true,
		},
		{
			name: "int oneof passes",
			target: &struct {
				Mode int `validate:"oneof=1 2 3"`
			}{Mode: 2},
			wantErr: false,
		},
		{
			name: "int oneof fails",
			target: &struct {
				Mode int `validate:"oneof=1 2 3"`
			}{Mode: 5},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rv := reflect.ValueOf(tt.target).Elem()
			fields := extractFields(rv, "", nil)
			err := validate(tt.target, fields)

			if tt.wantErr && err == nil {
				t.Fatalf("expected validation error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if tt.wantErr {
				var ve *ValidationError
				if !errors.As(err, &ve) {
					t.Fatalf("expected *ValidationError, got %T", err)
				}
				if !containsRule(ve, "oneof=") {
					t.Error("expected oneof rule in errors")
				}
			}
		})
	}
}

func TestValidateCombinedRules(t *testing.T) {
	tests := []struct {
		name      string
		target    any
		wantErr   bool
		wantRules []string
	}{
		{
			name: "all rules pass",
			target: &struct {
				Port int `validate:"required,min=1,max=100"`
			}{Port: 50},
			wantErr: false,
		},
		{
			name: "required fails, min/max not checked independently",
			target: &struct {
				Port int `validate:"required,min=1,max=100"`
			}{Port: 0},
			wantErr:   true,
			wantRules: []string{"required", "min=1"},
		},
		{
			name: "value too high fails max",
			target: &struct {
				Port int `validate:"required,min=1,max=100"`
			}{Port: 200},
			wantErr:   true,
			wantRules: []string{"max=100"},
		},
		{
			name: "required and oneof combined pass",
			target: &struct {
				Level string `validate:"required,oneof=debug info warn error"`
			}{Level: "info"},
			wantErr: false,
		},
		{
			name: "required and oneof combined - empty fails both",
			target: &struct {
				Level string `validate:"required,oneof=debug info warn error"`
			}{Level: ""},
			wantErr:   true,
			wantRules: []string{"required"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rv := reflect.ValueOf(tt.target).Elem()
			fields := extractFields(rv, "", nil)
			err := validate(tt.target, fields)

			if tt.wantErr && err == nil {
				t.Fatalf("expected validation error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if tt.wantErr && tt.wantRules != nil {
				var ve *ValidationError
				if !errors.As(err, &ve) {
					t.Fatalf("expected *ValidationError, got %T", err)
				}
				for _, rule := range tt.wantRules {
					if !containsRule(ve, rule) {
						t.Errorf("expected rule %q in errors, got %v", rule, ve.Errors)
					}
				}
			}
		})
	}
}

func TestValidationErrorMultipleFields(t *testing.T) {
	type Config struct {
		Host     string `validate:"required"`
		Port     int    `validate:"required,min=1,max=65535"`
		LogLevel string `validate:"oneof=debug info warn error"`
	}

	cfg := &Config{
		Host:     "",    // fails required
		Port:     0,     // fails required and min=1
		LogLevel: "foo", // fails oneof
	}

	rv := reflect.ValueOf(cfg).Elem()
	fields := extractFields(rv, "", nil)
	err := validate(cfg, fields)

	if err == nil {
		t.Fatal("expected validation error, got nil")
	}

	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected *ValidationError, got %T", err)
	}

	// Should have errors from all three fields
	if len(ve.Errors) < 3 {
		t.Errorf("expected at least 3 field errors, got %d: %v", len(ve.Errors), ve.Errors)
	}

	// Verify error message lists all fields
	errMsg := ve.Error()
	if !containsSubstring(errMsg, "Host") {
		t.Errorf("error message should mention Host: %s", errMsg)
	}
	if !containsSubstring(errMsg, "Port") {
		t.Errorf("error message should mention Port: %s", errMsg)
	}
	if !containsSubstring(errMsg, "LogLevel") {
		t.Errorf("error message should mention LogLevel: %s", errMsg)
	}

	// Verify it unwraps to ErrValidation
	if !errors.Is(err, ErrValidation) {
		t.Error("expected error to wrap ErrValidation")
	}
}

func TestValidateNoRules(t *testing.T) {
	cfg := &struct {
		Name string
		Port int
	}{Name: "", Port: 0}

	rv := reflect.ValueOf(cfg).Elem()
	fields := extractFields(rv, "", nil)
	err := validate(cfg, fields)
	if err != nil {
		t.Fatalf("expected no error for struct without validate tags, got %v", err)
	}
}

func TestValidateNestedStruct(t *testing.T) {
	type DB struct {
		Host string `validate:"required"`
		Port int    `validate:"min=1,max=65535"`
	}
	type Config struct {
		DB DB
	}

	t.Run("nested fields pass", func(t *testing.T) {
		cfg := &Config{DB: DB{Host: "localhost", Port: 5432}}
		rv := reflect.ValueOf(cfg).Elem()
		fields := extractFields(rv, "", nil)
		err := validate(cfg, fields)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})

	t.Run("nested fields fail", func(t *testing.T) {
		cfg := &Config{DB: DB{Host: "", Port: 0}}
		rv := reflect.ValueOf(cfg).Elem()
		fields := extractFields(rv, "", nil)
		err := validate(cfg, fields)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		var ve *ValidationError
		if !errors.As(err, &ve) {
			t.Fatalf("expected *ValidationError, got %T", err)
		}
		// Should have errors for both DB.Host (required) and DB.Port (min=1)
		if len(ve.Errors) < 2 {
			t.Errorf("expected at least 2 errors, got %d: %v", len(ve.Errors), ve.Errors)
		}
		// Check field paths include the nested prefix
		foundHost := false
		foundPort := false
		for _, fe := range ve.Errors {
			if fe.Field == "DB.Host" {
				foundHost = true
			}
			if fe.Field == "DB.Port" {
				foundPort = true
			}
		}
		if !foundHost {
			t.Error("expected error for field DB.Host")
		}
		if !foundPort {
			t.Error("expected error for field DB.Port")
		}
	})
}

func TestFieldErrorString(t *testing.T) {
	fe := FieldError{
		Field:   "DB.Port",
		Rule:    "min=1",
		Message: "value 0 is less than minimum 1",
	}
	got := fe.Error()
	want := "field DB.Port: value 0 is less than minimum 1"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestValidationErrorString(t *testing.T) {
	ve := &ValidationError{
		Errors: []FieldError{
			{Field: "Host", Rule: "required", Message: "required field is empty"},
			{Field: "Port", Rule: "min=1", Message: "value 0 is less than minimum 1"},
		},
	}
	got := ve.Error()
	if !containsSubstring(got, "validation failed:") {
		t.Errorf("expected 'validation failed:' prefix, got %q", got)
	}
	if !containsSubstring(got, "Host") || !containsSubstring(got, "Port") {
		t.Errorf("expected both field names in error, got %q", got)
	}
}

// containsRule checks if a ValidationError contains a FieldError with the given rule prefix.
func containsRule(ve *ValidationError, rulePrefix string) bool {
	for _, fe := range ve.Errors {
		if fe.Rule == rulePrefix || (len(rulePrefix) > 0 && len(fe.Rule) >= len(rulePrefix) && fe.Rule[:len(rulePrefix)] == rulePrefix) {
			return true
		}
	}
	return false
}

func containsSubstring(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || findSubstring(s, sub))
}

func findSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
