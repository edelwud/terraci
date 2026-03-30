package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/edelwud/terraci/pkg/config"
	"github.com/edelwud/terraci/pkg/plugin/plugintest"
)

func TestLogGenerateHint_GitHubConfig(t *testing.T) {
	cfg := loadInitTestConfig(t, "plugins:\n  github: {}\n")

	output := plugintest.CaptureLogOutput(t, func() {
		logGenerateHint(cfg)
	})

	if !strings.Contains(output, ".github/workflows/terraform.yml") {
		t.Fatalf("output = %q, want GitHub workflow hint", output)
	}
}

func TestLogGenerateHint_GitLabConfig(t *testing.T) {
	cfg := loadInitTestConfig(t, "plugins:\n  gitlab: {}\n")

	output := plugintest.CaptureLogOutput(t, func() {
		logGenerateHint(cfg)
	})

	if !strings.Contains(output, ".gitlab-ci.yml") {
		t.Fatalf("output = %q, want GitLab pipeline hint", output)
	}
}

func loadInitTestConfig(t *testing.T, pluginConfigYAML string) *config.Config {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, ".terraci.yaml")
	content := "structure:\n  pattern: \"{service}/{environment}/{region}/{module}\"\n" + pluginConfigYAML
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	return cfg
}
