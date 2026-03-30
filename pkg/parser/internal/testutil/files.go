package testutil

import (
	"os"
	"path/filepath"
	"testing"
)

func SetupTempModule(tb testing.TB, files map[string]string) string {
	tb.Helper()

	tmpDir := tb.TempDir()
	for name, content := range files {
		WriteFile(tb, tmpDir, name, content)
	}

	return tmpDir
}

func CreateTestModuleDir(tb testing.TB, root string, parts ...string) string {
	tb.Helper()

	path := filepath.Join(append([]string{root}, parts...)...)
	if err := os.MkdirAll(path, 0o755); err != nil {
		tb.Fatalf("mkdir %s: %v", path, err)
	}

	return path
}

func WriteFile(tb testing.TB, dir, filename, content string) {
	tb.Helper()

	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		tb.Fatalf("write %s: %v", path, err)
	}
}
