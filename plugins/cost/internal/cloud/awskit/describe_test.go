package awskit

import "testing"

func TestDescribeBuilder_ConditionalSteps(t *testing.T) {
	t.Parallel()

	got := NewDescribeBuilder().
		String("engine", "redis").
		StringIf(true, "volume_type", "gp3").
		StringIf(false, "tenancy", "dedicated").
		IntIf(true, "nodes", 3).
		IntIf(false, "replicas", 2).
		FloatIf(true, "memory_gib", 6.5, "%.1f").
		FloatIf(false, "storage_gib", 20, "%.0f").
		Map()

	if got["engine"] != "redis" {
		t.Fatalf("engine = %q, want %q", got["engine"], "redis")
	}
	if got["volume_type"] != "gp3" {
		t.Fatalf("volume_type = %q, want %q", got["volume_type"], "gp3")
	}
	if got["nodes"] != "3" {
		t.Fatalf("nodes = %q, want %q", got["nodes"], "3")
	}
	if got["memory_gib"] != "6.5" {
		t.Fatalf("memory_gib = %q, want %q", got["memory_gib"], "6.5")
	}
	if _, ok := got["tenancy"]; ok {
		t.Fatalf("tenancy should not be present: %#v", got)
	}
	if _, ok := got["replicas"]; ok {
		t.Fatalf("replicas should not be present: %#v", got)
	}
	if _, ok := got["storage_gib"]; ok {
		t.Fatalf("storage_gib should not be present: %#v", got)
	}
}
