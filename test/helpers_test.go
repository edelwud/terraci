package test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"go.yaml.in/yaml/v4"

	"github.com/edelwud/terraci/cmd/terraci/cmd"
	"github.com/edelwud/terraci/pkg/plugin/registry"

	// Register all built-in plugins via init()
	_ "github.com/edelwud/terraci/plugins/cost"
	_ "github.com/edelwud/terraci/plugins/git"
	_ "github.com/edelwud/terraci/plugins/github"
	_ "github.com/edelwud/terraci/plugins/gitlab"
	_ "github.com/edelwud/terraci/plugins/policy"
	_ "github.com/edelwud/terraci/plugins/summary"
	_ "github.com/edelwud/terraci/plugins/update"
)

// clearCIEnv neutralizes CI-specific environment variables so that
// DetectEnv() in provider plugins does not interfere with tests.
// Without this, running tests on GitHub Actions causes the github plugin
// to win provider resolution over the gitlab plugin configured in fixtures.
func clearCIEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{"GITHUB_ACTIONS", "GITLAB_CI", "CI_SERVER_URL"} {
		t.Setenv(key, "")
	}
}

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
	clearCIEnv(t)
	registry.ResetPlugins()

	t.Chdir(dir)

	rootCmd := cmd.NewRootCmd("test", "test-commit", "2024-01-01")
	rootCmd.SetArgs(args)

	return rootCmd.Execute()
}

// captureTerraCi executes a terraci command and captures os.Stdout output.
// This is needed for commands that write via fmt.Print (version, schema, generate without -o).
func captureTerraCi(t *testing.T, dir string, args ...string) (string, error) {
	t.Helper()
	clearCIEnv(t)
	registry.ResetPlugins()

	t.Chdir(dir)

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

// parseYAML parses YAML output into a generic map for structural validation.
// Strips terraci header comments before parsing.
func parseYAML(t *testing.T, data string) map[string]any {
	t.Helper()
	lines := strings.Split(data, "\n")
	var yamlLines []string
	for _, line := range lines {
		if !strings.HasPrefix(line, "#") {
			yamlLines = append(yamlLines, line)
		}
	}

	var result map[string]any
	if err := yaml.Unmarshal([]byte(strings.Join(yamlLines, "\n")), &result); err != nil {
		t.Fatalf("failed to parse YAML: %v", err)
	}
	return result
}

// assertContains checks that output contains a substring.
func assertContains(t *testing.T, output, substr string) {
	t.Helper()
	if !strings.Contains(output, substr) {
		t.Errorf("output should contain %q, got:\n%s", substr, truncate(output, 500))
	}
}

// assertNotContains checks that output does NOT contain a substring.
func assertNotContains(t *testing.T, output, substr string) {
	t.Helper()
	if strings.Contains(output, substr) {
		t.Errorf("output should NOT contain %q", substr)
	}
}

func truncate(s string, maxLen int) string {
	if len(s) > maxLen {
		return s[:maxLen] + "..."
	}
	return s
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
