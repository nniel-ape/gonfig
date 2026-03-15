package gonfig

import (
	"reflect"
	"testing"
	"time"
)

func TestSetFieldValue_String(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{"non-empty", "hello", "hello"},
		{"empty", "", ""},
		{"spaces", "  spaces  ", "  spaces  "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var s string
			v := reflect.ValueOf(&s).Elem()
			if err := setFieldValue(v, tt.raw); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if s != tt.want {
				t.Errorf("got %q, want %q", s, tt.want)
			}
		})
	}
}

func TestSetFieldValue_Int(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    int
		wantErr bool
	}{
		{"positive", "42", 42, false},
		{"zero", "0", 0, false},
		{"negative", "-10", -10, false},
		{"invalid", "abc", 0, true},
		{"float string", "3.14", 0, true},
		{"empty", "", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var n int
			v := reflect.ValueOf(&n).Elem()
			err := setFieldValue(v, tt.raw)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if n != tt.want {
				t.Errorf("got %d, want %d", n, tt.want)
			}
		})
	}
}

func TestSetFieldValue_Int64(t *testing.T) {
	var n int64
	v := reflect.ValueOf(&n).Elem()
	if err := setFieldValue(v, "9999999999"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 9999999999 {
		t.Errorf("got %d, want 9999999999", n)
	}
}

func TestSetFieldValue_Float64(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    float64
		wantErr bool
	}{
		{"positive", "3.14", 3.14, false},
		{"zero", "0", 0, false},
		{"negative", "-2.5", -2.5, false},
		{"integer", "42", 42.0, false},
		{"invalid", "abc", 0, true},
		{"empty", "", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var f float64
			v := reflect.ValueOf(&f).Elem()
			err := setFieldValue(v, tt.raw)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if f != tt.want {
				t.Errorf("got %f, want %f", f, tt.want)
			}
		})
	}
}

func TestSetFieldValue_Bool(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    bool
		wantErr bool
	}{
		{"true", "true", true, false},
		{"false", "false", false, false},
		{"1", "1", true, false},
		{"0", "0", false, false},
		{"TRUE", "TRUE", true, false},
		{"invalid", "yes", false, true},
		{"empty", "", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var b bool
			v := reflect.ValueOf(&b).Elem()
			err := setFieldValue(v, tt.raw)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if b != tt.want {
				t.Errorf("got %v, want %v", b, tt.want)
			}
		})
	}
}

func TestSetFieldValue_Duration(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    time.Duration
		wantErr bool
	}{
		{"seconds", "5s", 5 * time.Second, false},
		{"milliseconds", "100ms", 100 * time.Millisecond, false},
		{"minutes", "2m30s", 2*time.Minute + 30*time.Second, false},
		{"invalid", "abc", 0, true},
		{"empty", "", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var d time.Duration
			v := reflect.ValueOf(&d).Elem()
			err := setFieldValue(v, tt.raw)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if d != tt.want {
				t.Errorf("got %v, want %v", d, tt.want)
			}
		})
	}
}

func TestSetFieldValue_StringSlice(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want []string
	}{
		{"multiple", "a,b,c", []string{"a", "b", "c"}},
		{"single", "one", []string{"one"}},
		{"with spaces", "a, b, c", []string{"a", "b", "c"}},
		{"empty", "", []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var s []string
			v := reflect.ValueOf(&s).Elem()
			if err := setFieldValue(v, tt.raw); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(s, tt.want) {
				t.Errorf("got %v, want %v", s, tt.want)
			}
		})
	}
}

func TestSetFieldValue_IntSlice(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    []int
		wantErr bool
	}{
		{"multiple", "1,2,3", []int{1, 2, 3}, false},
		{"single", "42", []int{42}, false},
		{"with spaces", "1, 2, 3", []int{1, 2, 3}, false},
		{"empty", "", []int{}, false},
		{"invalid element", "1,abc,3", nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var s []int
			v := reflect.ValueOf(&s).Elem()
			err := setFieldValue(v, tt.raw)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(s, tt.want) {
				t.Errorf("got %v, want %v", s, tt.want)
			}
		})
	}
}

func TestSetFieldValue_UnsupportedType(t *testing.T) {
	var c complex128
	v := reflect.ValueOf(&c).Elem()
	err := setFieldValue(v, "1+2i")
	if err == nil {
		t.Fatal("expected error for unsupported type, got nil")
	}
}
