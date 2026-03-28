package test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/edelwud/terraci/cmd/terraci/cmd"

	// Register all built-in plugins via init()
	_ "github.com/edelwud/terraci/plugins/cost"
	_ "github.com/edelwud/terraci/plugins/git"
	_ "github.com/edelwud/terraci/plugins/github"
	_ "github.com/edelwud/terraci/plugins/gitlab"
	_ "github.com/edelwud/terraci/plugins/policy"
	_ "github.com/edelwud/terraci/plugins/summary"
)

func testdataDir(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to get caller info")
	}
	return filepath.Join(filepath.Dir(filename), "testdata")
}

func fixtureDir(t *testing.T, name string) string {
	t.Helper()
	return filepath.Join(testdataDir(t), name)
}

// runTerraCi executes a terraci command in the given directory and returns any error.
// For commands that write to stdout (generate, graph, schema, version), use
// the -o flag or captureTerraCi instead.
func runTerraCi(t *testing.T, dir string, args ...string) error {
	t.Helper()

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	if chErr := os.Chdir(dir); chErr != nil {
		t.Fatalf("failed to chdir to %s: %v", dir, chErr)
	}
	defer os.Chdir(origDir) //nolint:errcheck // best-effort restore

	rootCmd := cmd.NewRootCmd("test", "test-commit", "2024-01-01")
	rootCmd.SetArgs(args)

	return rootCmd.Execute()
}

// captureTerraCi executes a terraci command and captures os.Stdout output.
// This is needed for commands that write via fmt.Print (version, schema, generate without -o).
func captureTerraCi(t *testing.T, dir string, args ...string) (string, error) {
	t.Helper()

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	if chErr := os.Chdir(dir); chErr != nil {
		t.Fatalf("failed to chdir to %s: %v", dir, chErr)
	}
	defer os.Chdir(origDir) //nolint:errcheck // best-effort restore

	// Create a temp file to capture stdout
	tmpFile, tmpErr := os.CreateTemp(t.TempDir(), "stdout-*.txt")
	if tmpErr != nil {
		t.Fatalf("failed to create temp file: %v", tmpErr)
	}
	tmpPath := tmpFile.Name()

	// Redirect os.Stdout
	origStdout := os.Stdout
	os.Stdout = tmpFile

	rootCmd := cmd.NewRootCmd("test", "test-commit", "2024-01-01")
	rootCmd.SetArgs(args)
	execErr := rootCmd.Execute()

	// Restore stdout
	os.Stdout = origStdout
	_ = tmpFile.Close()

	// Read captured output
	data, readErr := os.ReadFile(tmpPath)
	if readErr != nil {
		t.Fatalf("failed to read captured output: %v", readErr)
	}

	return string(data), execErr
}

// copyFixtureToTemp copies a fixture directory to a temp dir so tests can modify it freely.
func copyFixtureToTemp(t *testing.T, name string) string {
	t.Helper()
	src := fixtureDir(t, name)
	dst := t.TempDir()

	err := filepath.Walk(src, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, relErr := filepath.Rel(src, path)
		if relErr != nil {
			return relErr
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		return os.WriteFile(target, data, 0o644)
	})
	if err != nil {
		t.Fatalf("failed to copy fixture %s: %v", name, err)
	}

	return dst
}

// writeConfig writes a minimal .terraci.yaml config to the given directory.
func writeConfig(t *testing.T, dir string) {
	t.Helper()
	cfg := `structure:
  pattern: "{service}/{environment}/{region}/{module}"
plugins:
  gitlab:
    terraform_binary: terraform
    image:
      name: hashicorp/terraform:1.6
  summary: {}
`
	if err := os.WriteFile(filepath.Join(dir, ".terraci.yaml"), []byte(cfg), 0o600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}
}
