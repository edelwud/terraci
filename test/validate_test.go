package test

import (
	"strings"
	"testing"
)

func TestValidate_ValidProject(t *testing.T) {
	dir := fixtureDir(t, "basic")
	// validate outputs to log (stderr), not stdout, so we cannot capture output in-process.
	// The key assertion is that validation succeeds (no error returned), which confirms
	// all modules were found, parsed, and the dependency graph has no cycles.
	err := runTerraCi(t, dir, "validate")
	if err != nil {
		t.Fatalf("validate failed: %v", err)
	}
}

func TestValidate_InvalidDir(t *testing.T) {
	dir := t.TempDir() // empty dir, no modules
	err := runTerraCi(t, dir, "validate")
	if err == nil {
		t.Fatal("expected error for empty directory with no modules")
	}
	// workflow.Run returns NoModulesError for empty directories
	if !strings.Contains(err.Error(), "no modules found") {
		t.Errorf("expected 'no modules found' error, got: %v", err)
	}
}
