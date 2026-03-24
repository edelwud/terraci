package policy

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/edelwud/terraci/pkg/config"
)

func TestNewSource_Path(t *testing.T) {
	cfg := config.PolicySource{Path: "./policies"}
	src, err := NewSource(cfg)
	if err != nil {
		t.Fatalf("NewSource() error = %v", err)
	}

	pathSrc, ok := src.(*PathSource)
	if !ok {
		t.Fatal("expected PathSource")
	}
	if pathSrc.Path != "./policies" {
		t.Errorf("Path = %v, want %v", pathSrc.Path, "./policies")
	}
}

func TestNewSource_Git(t *testing.T) {
	cfg := config.PolicySource{
		Git: "https://github.com/example/policies.git",
		Ref: "main",
	}
	src, err := NewSource(cfg)
	if err != nil {
		t.Fatalf("NewSource() error = %v", err)
	}

	gitSrc, ok := src.(*GitSource)
	if !ok {
		t.Fatal("expected GitSource")
	}
	if gitSrc.URL != "https://github.com/example/policies.git" {
		t.Errorf("URL = %v, want %v", gitSrc.URL, "https://github.com/example/policies.git")
	}
	if gitSrc.Ref != "main" {
		t.Errorf("Ref = %v, want %v", gitSrc.Ref, "main")
	}
}

func TestNewSource_OCI(t *testing.T) {
	cfg := config.PolicySource{OCI: "oci://ghcr.io/example/policies:v1"}
	src, err := NewSource(cfg)
	if err != nil {
		t.Fatalf("NewSource() error = %v", err)
	}

	ociSrc, ok := src.(*OCISource)
	if !ok {
		t.Fatal("expected OCISource")
	}
	if ociSrc.URL != "oci://ghcr.io/example/policies:v1" {
		t.Errorf("URL = %v, want %v", ociSrc.URL, "oci://ghcr.io/example/policies:v1")
	}
}

func TestNewSource_Unknown(t *testing.T) {
	cfg := config.PolicySource{} // empty, no type
	_, err := NewSource(cfg)
	if err == nil {
		t.Error("expected error for unknown source type")
	}
}

func TestNewPuller(t *testing.T) {
	cfg := &config.PolicyConfig{
		Sources: []config.PolicySource{
			{Path: "./policies"},
		},
		CacheDir: ".cache/policies",
	}

	puller, err := NewPuller(cfg, "/root")
	if err != nil {
		t.Fatalf("NewPuller() error = %v", err)
	}

	if puller == nil {
		t.Fatal("NewPuller() returned nil")
	}
	if len(puller.sources) != 1 {
		t.Errorf("sources = %v, want 1 element", len(puller.sources))
	}
	if puller.cacheDir != "/root/.cache/policies" {
		t.Errorf("cacheDir = %v, want %v", puller.cacheDir, "/root/.cache/policies")
	}
}

func TestNewPuller_NilConfig(t *testing.T) {
	_, err := NewPuller(nil, "/root")
	if err == nil {
		t.Error("expected error for nil config")
	}
}

func TestNewPuller_DefaultCacheDir(t *testing.T) {
	cfg := &config.PolicyConfig{
		Sources: []config.PolicySource{
			{Path: "./policies"},
		},
	}

	puller, err := NewPuller(cfg, "/root")
	if err != nil {
		t.Fatalf("NewPuller() error = %v", err)
	}

	if puller.cacheDir != "/root/.terraci/policies" {
		t.Errorf("cacheDir = %v, want %v", puller.cacheDir, "/root/.terraci/policies")
	}
}

func TestNewPuller_ResolvesRelativePaths(t *testing.T) {
	cfg := &config.PolicyConfig{
		Sources: []config.PolicySource{
			{Path: "policies"},
		},
	}

	puller, err := NewPuller(cfg, "/root")
	if err != nil {
		t.Fatalf("NewPuller() error = %v", err)
	}

	pathSrc, ok := puller.sources[0].(*PathSource)
	if !ok {
		t.Fatal("expected PathSource")
	}
	if pathSrc.Path != "/root/policies" {
		t.Errorf("Path = %v, want %v", pathSrc.Path, "/root/policies")
	}
}

func TestPuller_Pull_PathSource(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a policy directory
	policyDir := filepath.Join(tmpDir, "policies")
	if err := os.MkdirAll(policyDir, 0o755); err != nil {
		t.Fatalf("failed to create policy dir: %v", err)
	}

	cfg := &config.PolicyConfig{
		Sources: []config.PolicySource{
			{Path: policyDir},
		},
		CacheDir: filepath.Join(tmpDir, "cache"),
	}

	puller, err := NewPuller(cfg, tmpDir)
	if err != nil {
		t.Fatalf("NewPuller() error = %v", err)
	}

	dirs, err := puller.Pull(context.Background())
	if err != nil {
		t.Fatalf("Pull() error = %v", err)
	}

	// Path sources should return the original path directly
	if len(dirs) != 1 {
		t.Fatalf("expected 1 dir, got %d", len(dirs))
	}
	if dirs[0] != policyDir {
		t.Errorf("dir = %v, want %v", dirs[0], policyDir)
	}
}

func TestPuller_CacheDir(t *testing.T) {
	cfg := &config.PolicyConfig{
		Sources:  []config.PolicySource{{Path: "./policies"}},
		CacheDir: "/custom/cache",
	}

	puller, err := NewPuller(cfg, "/root")
	if err != nil {
		t.Fatalf("NewPuller() error = %v", err)
	}

	if puller.CacheDir() != "/custom/cache" {
		t.Errorf("CacheDir() = %v, want %v", puller.CacheDir(), "/custom/cache")
	}
}

func TestPathSource_String(t *testing.T) {
	src := &PathSource{Path: "/path/to/policies"}
	if src.String() != "path:/path/to/policies" {
		t.Errorf("String() = %v, want %v", src.String(), "path:/path/to/policies")
	}
}

func TestGitSource_String(t *testing.T) {
	tests := []struct {
		name     string
		source   GitSource
		expected string
	}{
		{
			name:     "without ref",
			source:   GitSource{URL: "https://github.com/example/repo.git"},
			expected: "git:https://github.com/example/repo.git",
		},
		{
			name:     "with ref",
			source:   GitSource{URL: "https://github.com/example/repo.git", Ref: "main"},
			expected: "git:https://github.com/example/repo.git@main",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.source.String(); got != tt.expected {
				t.Errorf("String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestOCISource_String(t *testing.T) {
	src := &OCISource{URL: "oci://ghcr.io/example/policies:v1"}
	expected := "oci:oci://ghcr.io/example/policies:v1"
	if src.String() != expected {
		t.Errorf("String() = %v, want %v", src.String(), expected)
	}
}
