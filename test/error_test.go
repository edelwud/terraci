package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestError_NoConfigExplicitPath(t *testing.T) {
	dir := t.TempDir()
	// Explicit -c pointing to a non-existent file should fail
	err := runTerraCi(t, dir, "generate", "-c", filepath.Join(dir, "nonexistent.yaml"))
	if err == nil {
		t.Fatal("expected error when config file doesn't exist")
	}
}

func TestError_InvalidYAMLConfig(t *testing.T) {
	dir := t.TempDir()
	// Write invalid YAML
	if err := os.WriteFile(filepath.Join(dir, ".terraci.yaml"), []byte("{{invalid yaml"), 0o600); err != nil {
		t.Fatal(err)
	}

	err := runTerraCi(t, dir, "generate")
	if err == nil {
		t.Fatal("expected error for invalid YAML config")
	}
}

func TestError_NonExistentDir(t *testing.T) {
	err := runTerraCi(t, t.TempDir(), "generate", "-d", "/nonexistent/path/12345")
	if err == nil {
		t.Fatal("expected error for non-existent directory")
	}
}

func TestError_NonExistentConfigFile(t *testing.T) {
	err := runTerraCi(t, t.TempDir(), "generate", "-c", "/nonexistent/config.yaml")
	if err == nil {
		t.Fatal("expected error for non-existent config file")
	}
}

func TestError_InvalidLogLevel(t *testing.T) {
	dir := fixtureDir(t, "basic")
	err := runTerraCi(t, dir, "generate", "-l", "banana")
	if err == nil {
		t.Fatal("expected error for invalid log level")
	}
	if !strings.Contains(err.Error(), "invalid log level") {
		t.Errorf("expected 'invalid log level' in error, got: %v", err)
	}
}
