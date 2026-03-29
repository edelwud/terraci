package plugin

import "testing"

func TestAppContext_Freeze(t *testing.T) {
	ctx := &AppContext{
		WorkDir: "/tmp",
		Version: "1.0",
	}

	if ctx.IsFrozen() {
		t.Error("should not be frozen initially")
	}

	ctx.Freeze()

	if !ctx.IsFrozen() {
		t.Error("should be frozen after Freeze()")
	}
}

func TestAppContext_ReportRegistry(t *testing.T) {
	r := NewReportRegistry()
	ctx := &AppContext{
		Reports: r,
	}

	if ctx.Reports == nil {
		t.Fatal("Reports should not be nil")
	}

	// Verify it's the same registry
	if ctx.Reports != r {
		t.Error("Reports should be the same registry instance")
	}
}
