package update

import (
	"testing"

	"github.com/edelwud/terraci/pkg/plugin/initwiz"
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
	if f.Key != "update.enabled" {
		t.Errorf("field.Key = %q, want %q", f.Key, "update.enabled")
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
	wantKeys := []string{"update.target", "update.bump", "update.pipeline"}
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
	stateEnabled.Set("update.enabled", true)
	if !showWhen(stateEnabled) {
		t.Error("ShowWhen should return true when update.enabled=true")
	}

	stateDisabled := initwiz.NewStateMap()
	stateDisabled.Set("update.enabled", false)
	if showWhen(stateDisabled) {
		t.Error("ShowWhen should return false when update.enabled=false")
	}
}

func TestPlugin_BuildInitConfig_Enabled(t *testing.T) {
	p := newTestPlugin(t)
	state := initwiz.NewStateMap()
	state.Set("update.enabled", true)

	contrib := p.BuildInitConfig(state)
	if contrib == nil {
		t.Fatal("BuildInitConfig() returned nil, want non-nil for enabled state")
	}
	if contrib.PluginKey != "update" {
		t.Errorf("PluginKey = %q, want %q", contrib.PluginKey, "update")
	}
	enabled, ok := contrib.Config["enabled"]
	if !ok {
		t.Fatal("Config missing 'enabled' key")
	}
	if enabled != true {
		t.Errorf("Config[enabled] = %v, want true", enabled)
	}
}

func TestPlugin_BuildInitConfig_Disabled(t *testing.T) {
	p := newTestPlugin(t)
	state := initwiz.NewStateMap()
	state.Set("update.enabled", false)

	contrib := p.BuildInitConfig(state)
	if contrib != nil {
		t.Errorf("BuildInitConfig() = %v, want nil for disabled state", contrib)
	}
}

func TestPlugin_BuildInitConfig_NotSet(t *testing.T) {
	p := newTestPlugin(t)
	state := initwiz.NewStateMap()

	contrib := p.BuildInitConfig(state)
	if contrib != nil {
		t.Errorf("BuildInitConfig() = %v, want nil for unset state", contrib)
	}
}

func TestPlugin_BuildInitConfig_NonDefaultTarget(t *testing.T) {
	p := newTestPlugin(t)
	state := initwiz.NewStateMap()
	state.Set("update.enabled", true)
	state.Set("update.target", "modules")

	contrib := p.BuildInitConfig(state)
	if contrib == nil {
		t.Fatal("BuildInitConfig() returned nil")
	}
	if contrib.Config["target"] != "modules" {
		t.Errorf("Config[target] = %v, want 'modules'", contrib.Config["target"])
	}
}

func TestPlugin_BuildInitConfig_NonDefaultBump(t *testing.T) {
	p := newTestPlugin(t)
	state := initwiz.NewStateMap()
	state.Set("update.enabled", true)
	state.Set("update.bump", "patch")

	contrib := p.BuildInitConfig(state)
	if contrib == nil {
		t.Fatal("BuildInitConfig() returned nil")
	}
	if contrib.Config["bump"] != "patch" {
		t.Errorf("Config[bump] = %v, want 'patch'", contrib.Config["bump"])
	}
}

func TestPlugin_BuildInitConfig_Pipeline(t *testing.T) {
	p := newTestPlugin(t)
	state := initwiz.NewStateMap()
	state.Set("update.enabled", true)
	state.Set("update.pipeline", true)

	contrib := p.BuildInitConfig(state)
	if contrib == nil {
		t.Fatal("BuildInitConfig() returned nil")
	}
	if contrib.Config["pipeline"] != true {
		t.Errorf("Config[pipeline] = %v, want true", contrib.Config["pipeline"])
	}
}

func TestPlugin_BuildInitConfig_AllDefaults(t *testing.T) {
	p := newTestPlugin(t)
	state := initwiz.NewStateMap()
	state.Set("update.enabled", true)
	state.Set("update.target", "all")
	state.Set("update.bump", "minor")

	contrib := p.BuildInitConfig(state)
	if contrib == nil {
		t.Fatal("BuildInitConfig() returned nil")
	}
	// Default values should be omitted
	if _, ok := contrib.Config["target"]; ok {
		t.Error("Config should not contain 'target' when it's the default 'all'")
	}
	if _, ok := contrib.Config["bump"]; ok {
		t.Error("Config should not contain 'bump' when it's the default 'minor'")
	}
	if _, ok := contrib.Config["pipeline"]; ok {
		t.Error("Config should not contain 'pipeline' when not set to true")
	}
}
