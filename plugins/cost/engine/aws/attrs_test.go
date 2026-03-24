package aws

import "testing"

func TestGetStringAttr(t *testing.T) {
	t.Parallel()

	attrs := map[string]any{
		"name":      "web",
		"count":     42,
		"enabled":   true,
		"empty_str": "",
	}

	tests := []struct {
		key  string
		want string
	}{
		{"name", "web"},
		{"empty_str", ""},
		{"missing", ""},
		{"count", ""},   // wrong type → ""
		{"enabled", ""}, // wrong type → ""
	}

	for _, tt := range tests {
		if got := GetStringAttr(attrs, tt.key); got != tt.want {
			t.Errorf("GetStringAttr(%q) = %q, want %q", tt.key, got, tt.want)
		}
	}
}

func TestGetStringAttr_NilMap(t *testing.T) {
	t.Parallel()

	if got := GetStringAttr(nil, "key"); got != "" {
		t.Errorf("GetStringAttr(nil, key) = %q, want empty", got)
	}
}

func TestGetFloatAttr(t *testing.T) {
	t.Parallel()

	attrs := map[string]any{
		"f64":    3.14,
		"int":    42,
		"int64":  int64(99),
		"str":    "nope",
		"zero_f": 0.0,
	}

	tests := []struct {
		key  string
		want float64
	}{
		{"f64", 3.14},
		{"int", 42.0},
		{"int64", 99.0},
		{"str", 0},     // wrong type → 0
		{"missing", 0}, // missing → 0
		{"zero_f", 0.0},
	}

	for _, tt := range tests {
		if got := GetFloatAttr(attrs, tt.key); got != tt.want {
			t.Errorf("GetFloatAttr(%q) = %v, want %v", tt.key, got, tt.want)
		}
	}
}

func TestGetIntAttr(t *testing.T) {
	t.Parallel()

	attrs := map[string]any{
		"f64":   3.7,
		"int":   42,
		"int64": int64(99),
		"str":   "nope",
	}

	tests := []struct {
		key  string
		want int
	}{
		{"f64", 3}, // truncated
		{"int", 42},
		{"int64", 99},
		{"str", 0},
		{"missing", 0},
	}

	for _, tt := range tests {
		if got := GetIntAttr(attrs, tt.key); got != tt.want {
			t.Errorf("GetIntAttr(%q) = %d, want %d", tt.key, got, tt.want)
		}
	}
}

func TestGetBoolAttr(t *testing.T) {
	t.Parallel()

	attrs := map[string]any{
		"enabled":  true,
		"disabled": false,
		"str":      "true", // wrong type
		"num":      1,      // wrong type
	}

	tests := []struct {
		key  string
		want bool
	}{
		{"enabled", true},
		{"disabled", false},
		{"str", false},     // wrong type → false
		{"num", false},     // wrong type → false
		{"missing", false}, // missing → false
	}

	for _, tt := range tests {
		if got := GetBoolAttr(attrs, tt.key); got != tt.want {
			t.Errorf("GetBoolAttr(%q) = %v, want %v", tt.key, got, tt.want)
		}
	}
}
