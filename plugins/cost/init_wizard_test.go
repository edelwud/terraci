package cost

import (
	"testing"

	"github.com/edelwud/terraci/pkg/plugin"
)

func TestPlugin_InitGroups(t *testing.T) {
	p := newTestPlugin(t)
	groups := p.InitGroups()

	if len(groups) != 1 {
		t.Fatalf("InitGroups() returned %d groups, want 1", len(groups))
	}

	g := groups[0]
	if g.Title != "Cost Estimation" {
		t.Errorf("group.Title = %q, want %q", g.Title, "Cost Estimation")
	}
	if g.Category != plugin.CategoryFeature {
		t.Errorf("group.Category = %v, want CategoryFeature", g.Category)
	}
	if g.Order != initGroupOrder {
		t.Errorf("group.Order = %d, want %d", g.Order, initGroupOrder)
	}
	if len(g.Fields) != 1 {
		t.Fatalf("fields count = %d, want 1", len(g.Fields))
	}

	f := g.Fields[0]
	if f.Key != "cost.enabled" {
		t.Errorf("field.Key = %q, want %q", f.Key, "cost.enabled")
	}
	if f.Type != "bool" {
		t.Errorf("field.Type = %q, want %q", f.Type, "bool")
	}
	if f.Default != false {
		t.Errorf("field.Default = %v, want false", f.Default)
	}
}

func TestPlugin_BuildInitConfig_Enabled(t *testing.T) {
	p := newTestPlugin(t)
	state := plugin.NewStateMap()
	state.Set("cost.enabled", true)

	contrib := p.BuildInitConfig(state)

	if contrib == nil {
		t.Fatal("BuildInitConfig() returned nil, want non-nil for enabled state")
	}
	if contrib.PluginKey != "cost" {
		t.Errorf("PluginKey = %q, want %q", contrib.PluginKey, "cost")
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
	state := plugin.NewStateMap()
	state.Set("cost.enabled", false)

	contrib := p.BuildInitConfig(state)

	if contrib != nil {
		t.Errorf("BuildInitConfig() = %v, want nil for disabled state", contrib)
	}
}

func TestPlugin_BuildInitConfig_NotSet(t *testing.T) {
	p := newTestPlugin(t)
	state := plugin.NewStateMap()
	// No key set — Bool("cost.enabled") returns false

	contrib := p.BuildInitConfig(state)

	if contrib != nil {
		t.Errorf("BuildInitConfig() = %v, want nil for unset state", contrib)
	}
}
