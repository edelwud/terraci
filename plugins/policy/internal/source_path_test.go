package policyengine

import (
	"context"
	"os"
	"testing"
)

func TestPathSource_Pull_ValidDir(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pathsource-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(tmpDir) })

	src := &PathSource{Path: tmpDir}
	err = src.Pull(context.Background(), "")
	if err != nil {
		t.Errorf("expected no error for valid dir, got: %v", err)
	}
}

func TestPathSource_Pull_NonExistent(t *testing.T) {
	src := &PathSource{Path: "/nonexistent/path/that/does/not/exist"}
	err := src.Pull(context.Background(), "")
	if err == nil {
		t.Error("expected error for non-existent path, got nil")
	}
}

func TestPathSource_Pull_File(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "pathsource-file-*")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	tmpFile.Close()
	t.Cleanup(func() { os.Remove(tmpFile.Name()) })

	src := &PathSource{Path: tmpFile.Name()}
	err = src.Pull(context.Background(), "")
	if err == nil {
		t.Error("expected error for file path (not directory), got nil")
	}
}

func TestPathSource_StringRepresentation(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "simple path",
			path: "./policies",
			want: "path:./policies",
		},
		{
			name: "absolute path",
			path: "/etc/terraci/policies",
			want: "path:/etc/terraci/policies",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := &PathSource{Path: tt.path}
			got := src.String()
			if got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}
