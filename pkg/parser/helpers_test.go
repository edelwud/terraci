package parser

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/hcl/v2"
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

func TestParserSegmentsReturnsCopy(t *testing.T) {
	parser := NewParser([]string{"service", "environment"})

	segments := parser.Segments()
	segments[0] = "changed"

	got := parser.Segments()
	if got[0] != "service" {
		t.Fatalf("segments mutated through getter: got %q, want %q", got[0], "service")
	}
}

func TestParsedModuleTopLevelBlocksReturnsCopy(t *testing.T) {
	module := NewParsedModule("")
	module.SetTopLevelBlocks(map[string][]*hcl.Block{
		"locals": {{}},
	})

	blocks := module.TopLevelBlocks()
	delete(blocks, "locals")

	if _, ok := module.TopLevelBlocks()["locals"]; !ok {
		t.Fatal("top-level blocks mutated through getter")
	}
}
