package cost

import (
	"testing"

	"github.com/edelwud/terraci/pkg/plugin/initwiz"
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
	if g.Category != initwiz.CategoryFeature {
		t.Errorf("group.Category = %v, want CategoryFeature", g.Category)
	}
	if g.Order != initGroupOrder {
		t.Errorf("group.Order = %d, want %d", g.Order, initGroupOrder)
	}
	if len(g.Fields) != 1 {
		t.Fatalf("fields count = %d, want 1", len(g.Fields))
	}

	f := g.Fields[0]
	if f.Key != "cost.providers.aws.enabled" {
		t.Errorf("field.Key = %q, want %q", f.Key, "cost.providers.aws.enabled")
	}
	if f.Type != initwiz.FieldBool {
		t.Errorf("field.Type = %q, want %q", f.Type, initwiz.FieldBool)
	}
	if f.Default != false {
		t.Errorf("field.Default = %v, want false", f.Default)
	}
}

func TestPlugin_BuildInitConfig_Enabled(t *testing.T) {
	p := newTestPlugin(t)
	state := initwiz.NewStateMap()
	state.Set("cost.providers.aws.enabled", true)

	contrib := p.BuildInitConfig(state)

	if contrib == nil {
		t.Fatal("BuildInitConfig() returned nil, want non-nil for enabled state")
	}
	if contrib.PluginKey != "cost" {
		t.Errorf("PluginKey = %q, want %q", contrib.PluginKey, "cost")
	}
	providers, ok := contrib.Config["providers"].(map[string]any)
	if !ok {
		t.Fatal("Config missing 'providers' key")
	}
	awsCfg, ok := providers["aws"].(map[string]any)
	if !ok {
		t.Fatal("Config missing providers.aws")
	}
	if awsCfg["enabled"] != true {
		t.Errorf("Config[providers][aws][enabled] = %v, want true", awsCfg["enabled"])
	}
}

func TestPlugin_BuildInitConfig_Disabled(t *testing.T) {
	p := newTestPlugin(t)
	state := initwiz.NewStateMap()
	state.Set("cost.providers.aws.enabled", false)

	contrib := p.BuildInitConfig(state)

	if contrib != nil {
		t.Errorf("BuildInitConfig() = %v, want nil for disabled state", contrib)
	}
}

func TestPlugin_BuildInitConfig_NotSet(t *testing.T) {
	p := newTestPlugin(t)
	state := initwiz.NewStateMap()
	// No key set — Bool("cost.providers.aws.enabled") returns false

	contrib := p.BuildInitConfig(state)

	if contrib != nil {
		t.Errorf("BuildInitConfig() = %v, want nil for unset state", contrib)
	}
}
