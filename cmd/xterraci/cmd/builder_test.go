package cmd

import (
	"sort"
	"strings"
	"testing"
)

func TestBuilderValidate(t *testing.T) {
	tests := []struct {
		name    string
		builder Builder
		wantErr string
	}{
		{
			name:    "valid: no with or without",
			builder: Builder{},
		},
		{
			name:    "valid: without gitlab",
			builder: Builder{WithoutPlugins: []string{"gitlab"}},
		},
		{
			name:    "error: without nonexistent",
			builder: Builder{WithoutPlugins: []string{"nonexistent"}},
			wantErr: "unknown plugin",
		},
		{
			name:    "error: with bad module (no slash)",
			builder: Builder{WithPlugins: []string{"bad"}},
			wantErr: "invalid module path",
		},
		{
			name:    "valid: with proper module",
			builder: Builder{WithPlugins: []string{"github.com/foo/bar"}},
		},
		{
			name:    "valid: with module with version",
			builder: Builder{WithPlugins: []string{"github.com/foo/bar@v1.0.0"}},
		},
		{
			name:    "valid: with module with replacement",
			builder: Builder{WithPlugins: []string{"github.com/foo/bar=../local"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.builder.validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error %q should contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestResolvePlugins(t *testing.T) {
	t.Run("default: all built-in plugins", func(t *testing.T) {
		b := &Builder{}
		builtins, externals := b.resolvePlugins()

		if len(externals) != 0 {
			t.Errorf("expected no external plugins, got %d", len(externals))
		}

		// All built-in plugins should be present
		sort.Strings(builtins)
		for name, importPath := range BuiltinPlugins {
			found := false
			for _, imp := range builtins {
				if imp == importPath {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("missing built-in plugin %q (%s)", name, importPath)
			}
		}
	})

	t.Run("without gitlab: gitlab absent, others present", func(t *testing.T) {
		b := &Builder{WithoutPlugins: []string{"gitlab"}}
		builtins, _ := b.resolvePlugins()

		gitlabImport := BuiltinPlugins["gitlab"]
		for _, imp := range builtins {
			if imp == gitlabImport {
				t.Error("gitlab should be excluded")
			}
		}

		// Other plugins should still be present
		expectedCount := len(BuiltinPlugins) - 1
		if len(builtins) != expectedCount {
			t.Errorf("expected %d builtins, got %d", expectedCount, len(builtins))
		}
	})

	t.Run("with external plugin", func(t *testing.T) {
		b := &Builder{WithPlugins: []string{"github.com/foo/bar@v1.0.0"}}
		_, externals := b.resolvePlugins()

		if len(externals) != 1 {
			t.Fatalf("expected 1 external, got %d", len(externals))
		}
		if externals[0].Module != "github.com/foo/bar" {
			t.Errorf("module = %q, want github.com/foo/bar", externals[0].Module)
		}
		if externals[0].Version != "v1.0.0" {
			t.Errorf("version = %q, want v1.0.0", externals[0].Version)
		}
	})
}
