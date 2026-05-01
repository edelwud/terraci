package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/edelwud/terraci/pkg/config"
	"github.com/edelwud/terraci/pkg/plugin/initwiz"
	"github.com/edelwud/terraci/pkg/plugin/plugintest"
	"github.com/edelwud/terraci/pkg/plugin/registry"
)

func TestLogGenerateHint_GitHubConfig(t *testing.T) {
	cfg := loadInitTestConfig(t, "extensions:\n  github: {}\n")

	output := plugintest.CaptureLogOutput(t, func() {
		logGenerateHint(&App{Plugins: registry.New()}, cfg)
	})

	if !strings.Contains(output, ".github/workflows/terraform.yml") {
		t.Fatalf("output = %q, want GitHub workflow hint", output)
	}
}

func TestLogGenerateHint_GitLabConfig(t *testing.T) {
	cfg := loadInitTestConfig(t, "extensions:\n  gitlab: {}\n")

	output := plugintest.CaptureLogOutput(t, func() {
		logGenerateHint(&App{Plugins: registry.New()}, cfg)
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

func TestInitStateDefaults(t *testing.T) {
	state := initwiz.NewStateMap()
	initStateDefaults(registry.New(), state)

	if got := state.String("binary"); got != "terraform" {
		t.Fatalf("binary = %q, want terraform", got)
	}
	if got := state.Bool("plan_enabled"); !got {
		t.Fatal("plan_enabled should default to true")
	}
	if got := state.String("pattern"); got != config.DefaultConfig().Structure.Pattern {
		t.Fatalf("pattern = %q", got)
	}
	if got := state.Bool("summary.enabled"); !got {
		t.Fatal("summary.enabled should default to true")
	}
}
