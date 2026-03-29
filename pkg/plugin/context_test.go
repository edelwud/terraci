package plugin

import (
	"testing"

	"github.com/edelwud/terraci/pkg/config"
)

func TestAppContext_Freeze(t *testing.T) {
	ctx := NewAppContext(nil, "/tmp", "/tmp/.terraci", "1.0", nil)

	if ctx.IsFrozen() {
		t.Error("should not be frozen initially")
	}

	ctx.Freeze()

	if !ctx.IsFrozen() {
		t.Error("should be frozen after Freeze()")
	}

	ctx.Update(nil, "/other", "/other/.terraci", "2.0")
	if ctx.WorkDir() != "/tmp" {
		t.Error("frozen context should ignore framework updates")
	}
}

func TestAppContext_ReportRegistry(t *testing.T) {
	r := NewReportRegistry()
	ctx := NewAppContext(nil, "", "", "", r)

	if ctx.Reports() == nil {
		t.Fatal("Reports should not be nil")
	}

	// Verify it's the same registry
	if ctx.Reports() != r {
		t.Error("Reports should be the same registry instance")
	}
}

func TestAppContext_ConfigReturnsCopy(t *testing.T) {
	cfg := config.DefaultConfig()
	ctx := NewAppContext(cfg, "/tmp", "/tmp/.terraci", "1.0", nil)

	got := ctx.Config()
	got.ServiceDir = "changed"

	if ctx.Config().ServiceDir != ".terraci" {
		t.Error("Config() should return a defensive copy")
	}
}
