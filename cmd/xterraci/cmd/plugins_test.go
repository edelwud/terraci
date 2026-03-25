package cmd

import (
	"strings"
	"testing"
)

func TestBuiltinNames(t *testing.T) {
	names := builtinNames()
	if !strings.Contains(names, "gitlab") {
		t.Error("missing gitlab")
	}
	if !strings.Contains(names, "cost") {
		t.Error("missing cost")
	}
	if !strings.Contains(names, "github") {
		t.Error("missing github")
	}
	if !strings.Contains(names, "policy") {
		t.Error("missing policy")
	}
	if !strings.Contains(names, "git") {
		t.Error("missing git")
	}

	// Should be sorted (comma-separated)
	parts := strings.Split(names, ", ")
	for i := 1; i < len(parts); i++ {
		if parts[i-1] > parts[i] {
			t.Errorf("not sorted: %q > %q", parts[i-1], parts[i])
		}
	}
}

func TestValidateWithout(t *testing.T) {
	// Valid: known plugin
	if err := validateWithout([]string{"gitlab"}); err != nil {
		t.Errorf("unexpected error for known plugin: %v", err)
	}

	// Valid: empty
	if err := validateWithout(nil); err != nil {
		t.Errorf("unexpected error for nil: %v", err)
	}

	// Error: unknown plugin
	err := validateWithout([]string{"nonexistent"})
	if err == nil {
		t.Fatal("expected error for unknown plugin")
	}
	if !strings.Contains(err.Error(), "nonexistent") {
		t.Errorf("error should mention plugin name: %v", err)
	}
	if !strings.Contains(err.Error(), "available:") {
		t.Errorf("error should list available plugins: %v", err)
	}
}

func TestValidateWith(t *testing.T) {
	// Valid: proper module path
	if err := validateWith([]string{"github.com/foo/bar"}); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Valid: with version
	if err := validateWith([]string{"github.com/foo/bar@v1.0.0"}); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Valid: with replacement
	if err := validateWith([]string{"github.com/foo/bar=../local"}); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Valid: empty
	if err := validateWith(nil); err != nil {
		t.Errorf("unexpected error for nil: %v", err)
	}

	// Error: no slash
	err := validateWith([]string{"bad"})
	if err == nil {
		t.Fatal("expected error for module without slash")
	}
	if !strings.Contains(err.Error(), "bad") {
		t.Errorf("error should mention module: %v", err)
	}
}
