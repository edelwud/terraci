package cmd

import (
	"strings"
	"testing"

	"github.com/edelwud/terraci/pkg/plugin/plugintest"
)

func TestLogGenerateHint_GitHubConfig(t *testing.T) {
	output := plugintest.CaptureLogOutput(t, func() {
		logGenerateHint("terraci generate -o .github/workflows/terraform.yml")
	})

	if !strings.Contains(output, ".github/workflows/terraform.yml") {
		t.Fatalf("output = %q, want GitHub workflow hint", output)
	}
}

func TestLogGenerateHint_GitLabConfig(t *testing.T) {
	output := plugintest.CaptureLogOutput(t, func() {
		logGenerateHint("terraci generate -o .gitlab-ci.yml")
	})

	if !strings.Contains(output, ".gitlab-ci.yml") {
		t.Fatalf("output = %q, want GitLab pipeline hint", output)
	}
}
