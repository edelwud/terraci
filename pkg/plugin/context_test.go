package plugin

import (
	"testing"

	"github.com/edelwud/terraci/pkg/config"
)

func TestAppContext_Ensure_NilRefresh(t *testing.T) {
	ctx := &AppContext{Version: "test"}
	// Should not panic when Refresh is nil
	ctx.Ensure()
	if ctx.Version != "test" {
		t.Fatal("version should be preserved")
	}
}

func TestAppContext_Ensure_PopulatesFromRefresh(t *testing.T) {
	cfg := config.DefaultConfig()
	ctx := &AppContext{Version: "v1"}
	ctx.Refresh = func() {
		ctx.Config = cfg
		ctx.WorkDir = "/tmp/test"
	}

	// Before Ensure, Config is nil
	if ctx.Config != nil {
		t.Fatal("Config should be nil before Ensure")
	}

	ctx.Ensure()

	if ctx.Config != cfg {
		t.Fatal("Config should be set after Ensure")
	}
	if ctx.WorkDir != "/tmp/test" {
		t.Fatalf("WorkDir = %q, want /tmp/test", ctx.WorkDir)
	}
}

func TestAppContext_Ensure_SafeWithNilConfig(t *testing.T) {
	// Simulates the scenario where Commands() captures AppContext
	// before PersistentPreRunE loads config.
	ctx := &AppContext{Version: "v1"}
	var appConfig *config.Config

	ctx.Refresh = func() {
		ctx.Config = appConfig
		ctx.WorkDir = "/work"
	}

	// First call — config still nil (before PersistentPreRunE)
	ctx.Ensure()
	if ctx.Config != nil {
		t.Fatal("Config should be nil before config is loaded")
	}

	// Simulate PersistentPreRunE loading config
	appConfig = config.DefaultConfig()

	// Second call — config is now populated
	ctx.Ensure()
	if ctx.Config == nil {
		t.Fatal("Config should be set after config is loaded")
	}
	if ctx.Config.Structure.Pattern == "" {
		t.Fatal("Config should have default pattern")
	}
}
