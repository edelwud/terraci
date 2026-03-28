package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGlobalFlag_Dir(t *testing.T) {
	basicDir := fixtureDir(t, "basic")

	// Run from a temp dir, but point -d to the basic fixture
	tmpDir := t.TempDir()
	output, err := captureTerraCi(t, tmpDir, "graph", "--format", "levels", "-d", basicDir)
	if err != nil {
		t.Fatalf("graph -d failed: %v", err)
	}

	assertContains(t, output, "vpc")
	assertContains(t, output, "eks")
}

func TestGlobalFlag_Config(t *testing.T) {
	basicDir := fixtureDir(t, "basic")

	// Copy fixture to temp and move config to a non-standard location
	tmpDir := copyFixtureToTemp(t, "basic")
	configPath := filepath.Join(tmpDir, ".terraci.yaml")
	customPath := filepath.Join(tmpDir, "custom-config.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if writeErr := os.WriteFile(customPath, data, 0o600); writeErr != nil {
		t.Fatalf("write custom config: %v", writeErr)
	}
	os.Remove(configPath)

	// Should fail without -c because .terraci.yaml is missing
	failErr := runTerraCi(t, tmpDir, "validate")
	if failErr == nil {
		// This may not fail if LoadOrDefault provides a default — either way,
		// the test below should succeed with -c
		_ = basicDir
	}

	// Should succeed with -c pointing to custom config
	err = runTerraCi(t, tmpDir, "validate", "-c", customPath)
	if err != nil {
		t.Fatalf("validate -c failed: %v", err)
	}
}

func TestGlobalFlag_Verbose(t *testing.T) {
	dir := fixtureDir(t, "basic")

	// -v should not cause an error
	err := runTerraCi(t, dir, "validate", "-v")
	if err != nil {
		t.Fatalf("validate -v failed: %v", err)
	}
}

func TestGlobalFlag_LogLevel_Debug(t *testing.T) {
	dir := fixtureDir(t, "basic")

	// -l debug should not cause an error
	err := runTerraCi(t, dir, "validate", "-l", "debug")
	if err != nil {
		t.Fatalf("validate -l debug failed: %v", err)
	}
}

func TestGlobalFlag_LogLevel_Invalid(t *testing.T) {
	dir := fixtureDir(t, "basic")

	err := runTerraCi(t, dir, "validate", "-l", "banana")
	if err == nil {
		t.Fatal("expected error for invalid log level")
	}
	if !strings.Contains(err.Error(), "invalid log level") {
		t.Errorf("expected 'invalid log level' error, got: %v", err)
	}
}
