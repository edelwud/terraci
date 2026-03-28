package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInit_CI_GitLab(t *testing.T) {
	dir := t.TempDir()

	err := runTerraCi(t, dir, "init", "--ci", "--provider", "gitlab")
	if err != nil {
		t.Fatalf("init --ci failed: %v", err)
	}

	// Config file should exist
	configPath := filepath.Join(dir, ".terraci.yaml")
	data, readErr := os.ReadFile(configPath)
	if readErr != nil {
		t.Fatalf("config not created: %v", readErr)
	}

	content := string(data)

	// Should have structure pattern
	assertContains(t, content, "pattern:")

	// Should have gitlab plugin config
	assertContains(t, content, "gitlab:")

	// Should have terraform binary
	assertContains(t, content, "terraform_binary:")

	// Should have specific default values
	assertContains(t, content, "hashicorp/terraform:1.6")
	assertContains(t, content, "plan_enabled")
}

func TestInit_CI_GitHub(t *testing.T) {
	dir := t.TempDir()

	err := runTerraCi(t, dir, "init", "--ci", "--provider", "github")
	if err != nil {
		t.Fatalf("init --ci --provider github failed: %v", err)
	}

	data, readErr := os.ReadFile(filepath.Join(dir, ".terraci.yaml"))
	if readErr != nil {
		t.Fatalf("config not created: %v", readErr)
	}
	content := string(data)

	assertContains(t, content, "github:")
	assertContains(t, content, "runs_on:")
	assertContains(t, content, "ubuntu-latest")
}

func TestInit_ExistingConfig_NoForce(t *testing.T) {
	dir := t.TempDir()
	// Create existing config
	if err := os.WriteFile(filepath.Join(dir, ".terraci.yaml"), []byte("structure:\n  pattern: test"), 0o600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	// Should fail without --force
	err := runTerraCi(t, dir, "init", "--ci", "--provider", "gitlab")
	if err == nil {
		t.Fatal("expected error when config exists without --force")
	}
}

func TestInit_Force(t *testing.T) {
	dir := t.TempDir()
	oldContent := "structure:\n  pattern: \"{old}/{pattern}\"\n"
	if err := os.WriteFile(filepath.Join(dir, ".terraci.yaml"), []byte(oldContent), 0o600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	err := runTerraCi(t, dir, "init", "--ci", "--provider", "gitlab", "--force")
	if err != nil {
		t.Fatalf("init --force failed: %v", err)
	}

	data, readErr := os.ReadFile(filepath.Join(dir, ".terraci.yaml"))
	if readErr != nil {
		t.Fatalf("config not read: %v", readErr)
	}
	content := string(data)
	if strings.Contains(content, "{old}") {
		t.Error("config was not overwritten")
	}
	// New config should have real values
	assertContains(t, content, "gitlab:")
}

func TestInit_CustomPattern(t *testing.T) {
	dir := t.TempDir()

	err := runTerraCi(t, dir, "init", "--ci", "--provider", "gitlab", "--pattern", "{env}/{region}/{module}")
	if err != nil {
		t.Fatalf("init --pattern failed: %v", err)
	}

	data, readErr := os.ReadFile(filepath.Join(dir, ".terraci.yaml"))
	if readErr != nil {
		t.Fatalf("config not read: %v", readErr)
	}
	assertContains(t, string(data), "{env}/{region}/{module}")
}

func TestInit_CustomBinary(t *testing.T) {
	dir := t.TempDir()

	err := runTerraCi(t, dir, "init", "--ci", "--provider", "gitlab", "--binary", "tofu")
	if err != nil {
		t.Fatalf("init --binary tofu failed: %v", err)
	}

	data, readErr := os.ReadFile(filepath.Join(dir, ".terraci.yaml"))
	if readErr != nil {
		t.Fatalf("config not read: %v", readErr)
	}
	content := string(data)
	assertContains(t, content, "tofu")
	// Should NOT contain default terraform binary when custom is set
	assertNotContains(t, content, "hashicorp/terraform")
}
