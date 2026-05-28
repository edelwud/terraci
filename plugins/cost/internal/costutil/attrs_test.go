package costutil

import (
	"testing"

	"github.com/edelwud/terraci/plugins/cost/internal/resourcedef"
)

func raw(attrs map[string]any) resourcedef.RawAttrs {
	return resourcedef.NewRawAttrs(attrs)
}

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
		if got := GetStringAttr(raw(attrs), tt.key); got != tt.want {
			t.Errorf("GetStringAttr(%q) = %q, want %q", tt.key, got, tt.want)
		}
	}
}

func TestGetStringAttr_NilMap(t *testing.T) {
	t.Parallel()

	if got := GetStringAttr(resourcedef.EmptyRawAttrs(), "key"); got != "" {
		t.Errorf("GetStringAttr(resourcedef.EmptyRawAttrs(), key) = %q, want empty", got)
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
		if got := GetFloatAttr(raw(attrs), tt.key); got != tt.want {
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
		if got := GetIntAttr(raw(attrs), tt.key); got != tt.want {
			t.Errorf("GetIntAttr(%q) = %d, want %d", tt.key, got, tt.want)
		}
	}
}

func TestGetBoolAttr(t *testing.T) {
	t.Parallel()

	attrs := map[string]any{
		"enabled":  true,
		"disabled": false,
		"str":      "true",
		"num":      1,
		"bad":      "definitely",
	}

	tests := []struct {
		key  string
		want bool
	}{
		{"enabled", true},
		{"disabled", false},
		{"str", true},
		{"num", true},
		{"bad", false},
		{"missing", false}, // missing → false
	}

	for _, tt := range tests {
		if got := GetBoolAttr(raw(attrs), tt.key); got != tt.want {
			t.Errorf("GetBoolAttr(%q) = %v, want %v", tt.key, got, tt.want)
		}
	}
}

func TestGetStringSliceAttr(t *testing.T) {
	t.Parallel()

	attrs := map[string]any{
		"strings": []string{"a", "b"},
		"anys":    []any{"c", 7, "d"},
		"wrong":   "not-a-slice",
	}

	if got := GetStringSliceAttr(raw(attrs), "strings"); len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("GetStringSliceAttr(strings) = %#v, want [a b]", got)
	}
	if got := GetStringSliceAttr(raw(attrs), "anys"); len(got) != 2 || got[0] != "c" || got[1] != "d" {
		t.Fatalf("GetStringSliceAttr(anys) = %#v, want [c d]", got)
	}
	if got := GetStringSliceAttr(raw(attrs), "wrong"); got != nil {
		t.Fatalf("GetStringSliceAttr(wrong) = %#v, want nil", got)
	}
}

func TestGetFirstObjectAttr(t *testing.T) {
	t.Parallel()

	attrs := map[string]any{
		"objects": []any{map[string]any{"name": "first"}},
		"typed":   []map[string]any{{"name": "typed"}},
		"direct":  map[string]any{"name": "direct"},
		"empty":   []any{},
		"wrong":   []any{"nope"},
	}

	tests := []struct {
		key  string
		want string
	}{
		{"objects", "first"},
		{"typed", "typed"},
		{"direct", "direct"},
		{"empty", ""},
		{"wrong", ""},
		{"missing", ""},
	}

	for _, tt := range tests {
		got := GetFirstObjectAttr(raw(attrs), tt.key)
		if tt.want == "" {
			if !got.IsZero() {
				t.Fatalf("GetFirstObjectAttr(%q) = %#v, want nil", tt.key, got)
			}
			continue
		}
		if got.String("name") != tt.want {
			t.Fatalf("GetFirstObjectAttr(%q)[name] = %q, want %q", tt.key, got.String("name"), tt.want)
		}
	}
}
