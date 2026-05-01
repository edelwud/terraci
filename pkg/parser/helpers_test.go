package parser

import (
	"testing"

	"github.com/hashicorp/hcl/v2"

	"github.com/edelwud/terraci/pkg/parser/internal/testutil"
	parsermodel "github.com/edelwud/terraci/pkg/parser/model"
)

// setupTempModule creates a temporary module directory with the given files.
func setupTempModule(t *testing.T, files map[string]string) string {
	return testutil.SetupTempModule(t, files)
}

// createTestModuleDir creates nested directories and returns the leaf path.
func createTestModuleDir(t *testing.T, tmpDir string, parts ...string) string {
	return testutil.CreateTestModuleDir(t, tmpDir, parts...)
}

// writeTestFile writes content to a file in the given directory.
func writeTestFile(t *testing.T, dir, filename, content string) {
	testutil.WriteFile(t, dir, filename, content)
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
	module := parsermodel.NewParsedModule("")
	module.SetTopLevelBlocks(map[string][]*hcl.Block{
		"locals": {{}},
	})

	blocks := module.TopLevelBlocks()
	delete(blocks, "locals")

	if _, ok := module.TopLevelBlocks()["locals"]; !ok {
		t.Fatal("top-level blocks mutated through getter")
	}
}
