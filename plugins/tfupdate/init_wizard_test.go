package tfupdate

import (
	"testing"

	"github.com/edelwud/terraci/pkg/plugin/initwiz"
	tfupdateengine "github.com/edelwud/terraci/plugins/tfupdate/internal"
)

func TestPlugin_InitGroups(t *testing.T) {
	p := newTestPlugin(t)
	groups := p.InitGroups()

	if len(groups) != 2 {
		t.Fatalf("InitGroups() returned %d groups, want 2", len(groups))
	}

	// Group 0: Dependency Updates
	g0 := groups[0]
	if g0.Title != "Dependency Updates" {
		t.Errorf("group[0].Title = %q, want %q", g0.Title, "Dependency Updates")
	}
	if g0.Category != initwiz.CategoryFeature {
		t.Errorf("group[0].Category = %v, want CategoryFeature", g0.Category)
	}
	if g0.Order != initGroupOrder {
		t.Errorf("group[0].Order = %d, want %d", g0.Order, initGroupOrder)
	}
	if len(g0.Fields) != 1 {
		t.Fatalf("group[0] fields count = %d, want 1", len(g0.Fields))
	}
	f := g0.Fields[0]
	if f.Key != "tfupdate.enabled" {
		t.Errorf("field.Key = %q, want %q", f.Key, "tfupdate.enabled")
	}
	if f.Type != initwiz.FieldBool {
		t.Errorf("field.Type = %q, want %q", f.Type, initwiz.FieldBool)
	}
	if f.Default != false {
		t.Errorf("field.Default = %v, want false", f.Default)
	}

	// Group 1: Update Settings
	g1 := groups[1]
	if g1.Title != "Update Settings" {
		t.Errorf("group[1].Title = %q, want %q", g1.Title, "Update Settings")
	}
	if g1.Category != initwiz.CategoryDetail {
		t.Errorf("group[1].Category = %v, want CategoryDetail", g1.Category)
	}
	if len(g1.Fields) != 3 {
		t.Fatalf("group[1] fields count = %d, want 3", len(g1.Fields))
	}

	// Verify field keys
	keys := []string{g1.Fields[0].Key, g1.Fields[1].Key, g1.Fields[2].Key}
	wantKeys := []string{"tfupdate.target", "tfupdate.bump", "tfupdate.pipeline"}
	for i, want := range wantKeys {
		if keys[i] != want {
			t.Errorf("field[%d].Key = %q, want %q", i, keys[i], want)
		}
	}

	// Target has 3 options
	if len(g1.Fields[0].Options) != 3 {
		t.Errorf("target options = %d, want 3", len(g1.Fields[0].Options))
	}
	// Bump has 3 options
	if len(g1.Fields[1].Options) != 3 {
		t.Errorf("bump options = %d, want 3", len(g1.Fields[1].Options))
	}
}

func TestPlugin_InitGroups_ShowWhen(t *testing.T) {
	p := newTestPlugin(t)
	groups := p.InitGroups()
	showWhen := groups[1].ShowWhen

	if showWhen == nil {
		t.Fatal("group[1].ShowWhen is nil")
	}

	stateEnabled := initwiz.NewStateMap()
	stateEnabled.Set("tfupdate.enabled", true)
	if !showWhen(stateEnabled) {
		t.Error("ShowWhen should return true when tfupdate.enabled=true")
	}

	stateDisabled := initwiz.NewStateMap()
	stateDisabled.Set("tfupdate.enabled", false)
	if showWhen(stateDisabled) {
		t.Error("ShowWhen should return false when tfupdate.enabled=false")
	}
}

func TestPlugin_BuildInitConfig_Enabled(t *testing.T) {
	p := newTestPlugin(t)
	state := initwiz.NewStateMap()
	state.Set("tfupdate.enabled", true)

	contrib, err := p.BuildInitConfig(state)
	if err != nil {
		t.Fatalf("BuildInitConfig() error = %v", err)
	}
	if contrib == nil {
		t.Fatal("BuildInitConfig() returned nil, want non-nil for enabled state")
	}
	if contrib.PluginKey() != "tfupdate" {
		t.Errorf("PluginKey() = %q, want %q", contrib.PluginKey(), "tfupdate")
	}
	cfg := decodeInitConfig(t, contrib)
	if !cfg.Enabled {
		t.Errorf("Config.Enabled = false, want true")
	}
}

func TestPlugin_BuildInitConfig_Disabled(t *testing.T) {
	p := newTestPlugin(t)
	state := initwiz.NewStateMap()
	state.Set("tfupdate.enabled", false)

	contrib, err := p.BuildInitConfig(state)
	if err != nil {
		t.Fatalf("BuildInitConfig() error = %v", err)
	}
	if contrib != nil {
		t.Errorf("BuildInitConfig() = %v, want nil for disabled state", contrib)
	}
}

func TestPlugin_BuildInitConfig_NotSet(t *testing.T) {
	p := newTestPlugin(t)
	state := initwiz.NewStateMap()

	contrib, err := p.BuildInitConfig(state)
	if err != nil {
		t.Fatalf("BuildInitConfig() error = %v", err)
	}
	if contrib != nil {
		t.Errorf("BuildInitConfig() = %v, want nil for unset state", contrib)
	}
}

func TestPlugin_BuildInitConfig_NonDefaultTarget(t *testing.T) {
	p := newTestPlugin(t)
	state := initwiz.NewStateMap()
	state.Set("tfupdate.enabled", true)
	state.Set("tfupdate.target", "modules")

	contrib, err := p.BuildInitConfig(state)
	if err != nil {
		t.Fatalf("BuildInitConfig() error = %v", err)
	}
	if contrib == nil {
		t.Fatal("BuildInitConfig() returned nil")
	}
	cfg := decodeInitConfig(t, contrib)
	if cfg.Target != "modules" {
		t.Errorf("Config.Target = %v, want 'modules'", cfg.Target)
	}
}

func TestPlugin_BuildInitConfig_NonDefaultBump(t *testing.T) {
	p := newTestPlugin(t)
	state := initwiz.NewStateMap()
	state.Set("tfupdate.enabled", true)
	state.Set("tfupdate.bump", "patch")

	contrib, err := p.BuildInitConfig(state)
	if err != nil {
		t.Fatalf("BuildInitConfig() error = %v", err)
	}
	if contrib == nil {
		t.Fatal("BuildInitConfig() returned nil")
	}
	cfg := decodeInitConfig(t, contrib)
	if cfg.Policy.Bump != "patch" {
		t.Errorf("Config.Policy.Bump = %v, want 'patch'", cfg.Policy.Bump)
	}
}

func TestPlugin_BuildInitConfig_Pipeline(t *testing.T) {
	p := newTestPlugin(t)
	state := initwiz.NewStateMap()
	state.Set("tfupdate.enabled", true)
	state.Set("tfupdate.pipeline", true)

	contrib, err := p.BuildInitConfig(state)
	if err != nil {
		t.Fatalf("BuildInitConfig() error = %v", err)
	}
	if contrib == nil {
		t.Fatal("BuildInitConfig() returned nil")
	}
	cfg := decodeInitConfig(t, contrib)
	if !cfg.Pipeline {
		t.Errorf("Config.Pipeline = false, want true")
	}
}

func TestPlugin_BuildInitConfig_AllDefaults(t *testing.T) {
	p := newTestPlugin(t)
	state := initwiz.NewStateMap()
	state.Set("tfupdate.enabled", true)
	state.Set("tfupdate.target", "all")
	state.Set("tfupdate.bump", "minor")

	contrib, err := p.BuildInitConfig(state)
	if err != nil {
		t.Fatalf("BuildInitConfig() error = %v", err)
	}
	if contrib == nil {
		t.Fatal("BuildInitConfig() returned nil")
	}
	// Default values should be omitted
	cfg := decodeInitConfig(t, contrib)
	if cfg.Target != "" {
		t.Error("Config should not contain target when it's the default 'all'")
	}
	if cfg.Policy.Bump != "" {
		t.Error("Config should not contain policy.bump when bump is the default 'minor'")
	}
	if cfg.Pipeline {
		t.Error("Config should not contain pipeline when not set to true")
	}
}

func decodeInitConfig(tb testing.TB, contribution *initwiz.InitContribution) tfupdateengine.UpdateConfig {
	tb.Helper()
	var cfg tfupdateengine.UpdateConfig
	if err := contribution.DecodeConfig(&cfg); err != nil {
		tb.Fatalf("DecodeConfig() error = %v", err)
	}
	return cfg
}
