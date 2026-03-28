package test

import (
	"strings"
	"testing"
)

func TestVersion_ShowsVersion(t *testing.T) {
	dir := fixtureDir(t, "basic")

	output, err := captureTerraCi(t, dir, "version")
	if err != nil {
		t.Fatalf("version failed: %v", err)
	}

	if !strings.Contains(output, "terraci test") {
		t.Errorf("version output missing 'terraci test', got: %s", output)
	}
}

func TestVersion_ShowsPlugins(t *testing.T) {
	dir := fixtureDir(t, "basic")

	output, err := captureTerraCi(t, dir, "version")
	if err != nil {
		t.Fatalf("version failed: %v", err)
	}

	// Should list all registered plugins
	expectedPlugins := []string{"cost", "git", "github", "gitlab", "policy", "summary"}
	for _, name := range expectedPlugins {
		if !strings.Contains(output, name) {
			t.Errorf("version output missing plugin: %s\noutput: %s", name, output)
		}
	}
}

func TestVersion_ShowsCommitAndDate(t *testing.T) {
	dir := fixtureDir(t, "basic")

	output, err := captureTerraCi(t, dir, "version")
	if err != nil {
		t.Fatalf("version failed: %v", err)
	}

	if !strings.Contains(output, "test-commit") {
		t.Error("version output missing commit info")
	}
	if !strings.Contains(output, "2024-01-01") {
		t.Error("version output missing date info")
	}
}
