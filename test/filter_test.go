package test

import (
	"testing"
)

func TestFilter_ExcludeConsistency_Generate(t *testing.T) {
	dir := fixtureDir(t, "basic")

	output, err := captureTerraCi(t, dir, "generate", "--exclude", "**/eks")
	if err != nil {
		t.Fatalf("generate --exclude failed: %v", err)
	}
	assertNotContains(t, output, "eks")
	assertContains(t, output, "vpc")
}

func TestFilter_ExcludeConsistency_Graph(t *testing.T) {
	dir := fixtureDir(t, "basic")

	output, err := captureTerraCi(t, dir, "graph", "--format", "levels", "--exclude", "**/eks")
	if err != nil {
		t.Fatalf("graph --exclude failed: %v", err)
	}
	assertNotContains(t, output, "eks")
	assertContains(t, output, "vpc")
}

func TestFilter_ExcludeConsistency_Validate(t *testing.T) {
	dir := fixtureDir(t, "basic")

	err := runTerraCi(t, dir, "validate", "--exclude", "**/eks")
	if err != nil {
		t.Fatalf("validate --exclude failed: %v", err)
	}
}

func TestFilter_SegmentConsistency_Generate(t *testing.T) {
	dir := fixtureDir(t, "basic")

	output, err := captureTerraCi(t, dir, "generate", "--filter", "environment=prod")
	if err != nil {
		t.Fatalf("generate --filter failed: %v", err)
	}
	assertContains(t, output, "prod")
	// Verify no stage modules are present (check job names, not YAML keywords like "stages:")
	assertNotContains(t, output, "platform-stage-")
}

func TestFilter_SegmentConsistency_Graph(t *testing.T) {
	dir := fixtureDir(t, "basic")

	output, err := captureTerraCi(t, dir, "graph", "--format", "levels", "--filter", "environment=prod")
	if err != nil {
		t.Fatalf("graph --filter failed: %v", err)
	}
	assertContains(t, output, "prod")
	assertNotContains(t, output, "platform/stage")
}

func TestFilter_SegmentConsistency_Validate(t *testing.T) {
	dir := fixtureDir(t, "basic")

	err := runTerraCi(t, dir, "validate", "--filter", "environment=prod")
	if err != nil {
		t.Fatalf("validate --filter failed: %v", err)
	}
}

func TestFilter_MultipleSegments(t *testing.T) {
	dir := fixtureDir(t, "basic")

	output, err := captureTerraCi(t, dir, "generate", "--filter", "environment=prod", "--filter", "environment=stage")
	if err != nil {
		t.Fatalf("generate multi-filter failed: %v", err)
	}

	// Both prod and stage should be present
	assertContains(t, output, "prod")
	assertContains(t, output, "stage")
}

func TestFilter_CombinedExcludeAndSegment(t *testing.T) {
	dir := fixtureDir(t, "basic")

	output, err := captureTerraCi(t, dir, "generate", "--exclude", "**/eks", "--filter", "environment=prod")
	if err != nil {
		t.Fatalf("generate combined filters failed: %v", err)
	}

	// Only prod vpc (no eks because excluded, no stage because filtered)
	assertContains(t, output, "plan-platform-prod-eu-central-1-vpc")
	assertNotContains(t, output, "eks")
	// Check for stage in job names, not in YAML keywords
	assertNotContains(t, output, "platform-stage-")
}
