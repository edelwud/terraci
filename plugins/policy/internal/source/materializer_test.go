package source

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	policyengine "github.com/edelwud/terraci/plugins/policy/internal"
)

func TestMaterializer_PathSourceUsesOriginalDirectoryAndCreatesCache(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	policyDir := filepath.Join(root, "policies")
	if err := os.MkdirAll(policyDir, 0o755); err != nil {
		t.Fatal(err)
	}

	materializer, err := NewMaterializer(&policyengine.Config{
		Sources: []policyengine.SourceConfig{{Type: policyengine.SourceTypePath, Path: "policies"}},
	}, root, filepath.Join(root, ".terraci"))
	if err != nil {
		t.Fatalf("NewMaterializer() error = %v", err)
	}

	dirs, err := materializer.Materialize(context.Background(), "")
	if err != nil {
		t.Fatalf("Materialize() error = %v", err)
	}
	if len(dirs) != 1 || dirs[0] != policyDir {
		t.Fatalf("dirs = %v, want [%s]", dirs, policyDir)
	}
	if _, err := os.Stat(filepath.Join(root, ".terraci", "policies")); err != nil {
		t.Fatalf("cache dir was not created: %v", err)
	}
}

func TestNewSource_TypedSpecs(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	tests := []struct {
		name string
		cfg  policyengine.SourceConfig
		want string
	}{
		{name: "path", cfg: policyengine.SourceConfig{Type: policyengine.SourceTypePath, Path: "policies"}, want: "path:" + filepath.Join(root, "policies")},
		{name: "git", cfg: policyengine.SourceConfig{Type: policyengine.SourceTypeGit, URL: "https://github.com/org/policies.git", Ref: "main"}, want: "git:https://github.com/org/policies.git@main"},
		{name: "oci", cfg: policyengine.SourceConfig{Type: policyengine.SourceTypeOCI, URL: "oci://ghcr.io/org/policies:v1"}, want: "oci:oci://ghcr.io/org/policies:v1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			src, err := NewSource(tt.cfg, root)
			if err != nil {
				t.Fatalf("NewSource() error = %v", err)
			}
			if got := src.String(); got != tt.want {
				t.Fatalf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMaterializer_CacheDirOverrideIsWorkDirRelative(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	materializer, err := NewMaterializer(&policyengine.Config{
		Sources: []policyengine.SourceConfig{{Type: policyengine.SourceTypePath, Path: "policies"}},
	}, root, filepath.Join(root, ".terraci"))
	if err != nil {
		t.Fatalf("NewMaterializer() error = %v", err)
	}

	want := filepath.Join(root, "custom-cache")
	if got := materializer.CacheDir("custom-cache"); got != want {
		t.Fatalf("CacheDir() = %q, want %q", got, want)
	}
}
