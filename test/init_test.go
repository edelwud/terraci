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
	if !strings.Contains(content, "pattern:") {
		t.Error("missing pattern in config")
	}

	// Should have gitlab plugin config
	if !strings.Contains(content, "gitlab:") {
		t.Error("missing gitlab config")
	}

	// Should have terraform binary
	if !strings.Contains(content, "terraform_binary:") {
		t.Error("missing terraform_binary")
	}
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

	if !strings.Contains(content, "github:") {
		t.Error("missing github config")
	}
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
	if strings.Contains(string(data), "{old}") {
		t.Error("config was not overwritten")
	}
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
	if !strings.Contains(string(data), "{env}/{region}/{module}") {
		t.Error("custom pattern not in config")
	}
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
	if !strings.Contains(string(data), "tofu") {
		t.Error("custom binary not in config")
	}
}
