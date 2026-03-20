package parser

import (
	"os"
	"path/filepath"
	"testing"
)

// setupTempModule creates a temporary module directory with the given files.
func setupTempModule(t *testing.T, files map[string]string) string {
	t.Helper()
	tmpDir := t.TempDir()
	for name, content := range files {
		path := filepath.Join(tmpDir, name)
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	return tmpDir
}

// createTestModuleDir creates nested directories and returns the leaf path.
func createTestModuleDir(t *testing.T, tmpDir string, parts ...string) string {
	t.Helper()
	path := filepath.Join(append([]string{tmpDir}, parts...)...)
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	return path
}

// writeTestFile writes content to a file in the given directory.
func writeTestFile(t *testing.T, dir, filename, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, filename), []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", filename, err)
	}
}
