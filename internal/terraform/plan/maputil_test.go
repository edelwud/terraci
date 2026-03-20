package plan

import "testing"

func TestFormatValue(t *testing.T) {
	tests := []struct {
		input any
		want  string
	}{
		{nil, ""},
		{"hello", "hello"},
		{true, "true"},
		{false, "false"},
		{float64(42), "42"},
		{float64(3.14), "3.14"},
		{[]any{}, "[]"},
		{[]any{1, 2, 3}, "[3 items]"},
		{map[string]any{}, "{}"},
		{map[string]any{"a": 1, "b": 2}, "{2 keys}"},
	}

	for _, tt := range tests {
		if got := formatValue(tt.input); got != tt.want {
			t.Errorf("formatValue(%v) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestGetNestedValue(t *testing.T) {
	m := map[string]any{
		"a": "value_a",
		"b": map[string]any{
			"c": "value_c",
			"d": map[string]any{"e": "value_e"},
		},
	}

	tests := []struct {
		key  string
		want any
	}{
		{"a", "value_a"},
		{"b.c", "value_c"},
		{"b.d.e", "value_e"},
		{"nonexistent", nil},
		{"b.nonexistent", nil},
	}

	for _, tt := range tests {
		if got := getNestedValue(m, tt.key); got != tt.want {
			t.Errorf("getNestedValue(%q) = %v, want %v", tt.key, got, tt.want)
		}
	}
}

func TestGetNestedValue_NilMap(t *testing.T) {
	if got := getNestedValue(nil, "key"); got != nil {
		t.Errorf("getNestedValue(nil, key) = %v, want nil", got)
	}
}

func TestGetBool(t *testing.T) {
	m := map[string]any{
		"flag":   true,
		"nested": map[string]any{"deep": true},
		"str":    "not-a-bool",
	}

	if !getBool(m, "flag") {
		t.Error("getBool(flag) = false, want true")
	}
	if !getBool(m, "nested.deep") {
		t.Error("getBool(nested.deep) = false, want true")
	}
	if getBool(m, "str") {
		t.Error("getBool(str) = true, want false")
	}
	if getBool(m, "missing") {
		t.Error("getBool(missing) = true, want false")
	}
}

func TestToMap(t *testing.T) {
	m := map[string]any{"key": "val"}
	if got := toMap(m); got == nil || got["key"] != "val" {
		t.Errorf("toMap(map) = %v, want {key:val}", got)
	}
	if got := toMap(nil); got != nil {
		t.Errorf("toMap(nil) = %v, want nil", got)
	}
	if got := toMap("string"); got != nil {
		t.Errorf("toMap(string) = %v, want nil", got)
	}
}

func TestPathToString(t *testing.T) {
	tests := []struct {
		input []any
		want  string
	}{
		{[]any{"a", "b", "c"}, "a.b.c"},
		{[]any{"single"}, "single"},
		{[]any{}, ""},
		{[]any{"a", 42, "b"}, "a.b"}, // non-strings skipped
	}

	for _, tt := range tests {
		if got := pathToString(tt.input); got != tt.want {
			t.Errorf("pathToString(%v) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestCollectKeys(t *testing.T) {
	m := map[string]any{
		"a": "val",
		"b": map[string]any{"c": "val", "d": "val"},
	}

	keys := make(map[string]bool)
	collectKeys(m, "", keys)

	for _, want := range []string{"a", "b", "b.c", "b.d"} {
		if !keys[want] {
			t.Errorf("missing key %q", want)
		}
	}
	if len(keys) != 4 {
		t.Errorf("keys count = %d, want 4", len(keys))
	}
}
