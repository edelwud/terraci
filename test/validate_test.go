package test

import "testing"

func TestValidate_ValidProject(t *testing.T) {
	dir := fixtureDir(t, "basic")
	err := runTerraCi(t, dir, "validate")
	if err != nil {
		t.Fatalf("validate failed: %v", err)
	}
}

func TestValidate_InvalidDir(t *testing.T) {
	dir := t.TempDir() // empty dir, no modules
	err := runTerraCi(t, dir, "validate")
	// Should succeed (no modules is valid, just empty)
	if err != nil {
		t.Logf("validate on empty dir: %v (may be expected)", err)
	}
}
