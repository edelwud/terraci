package cost

import (
	"testing"

	"github.com/edelwud/terraci/pkg/plugin/initwiz"
	"github.com/edelwud/terraci/plugins/cost/internal/model"
)

func TestPlugin_InitGroups(t *testing.T) {
	p := newTestPlugin(t)
	groups := p.InitGroups()

	if len(groups) != 1 {
		t.Fatalf("InitGroups() returned %d groups, want 1", len(groups))
	}

	g := groups[0]
	if g.Title != costReportTitle {
		t.Errorf("group.Title = %q, want %q", g.Title, costReportTitle)
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
	if f.Key() != "cost.providers.aws.enabled" {
		t.Errorf("field.Key = %q, want %q", f.Key(), "cost.providers.aws.enabled")
	}
	if f.Type() != initwiz.FieldBool {
		t.Errorf("field.Type = %q, want %q", f.Type(), initwiz.FieldBool)
	}
}

func TestPlugin_BuildInitConfig_Enabled(t *testing.T) {
	p := newTestPlugin(t)
	state := initwiz.NewStateMap()
	providerEnabledKey("aws").Set(state, true)

	contrib, err := p.BuildInitConfig(state)
	if err != nil {
		t.Fatalf("BuildInitConfig() error = %v", err)
	}

	if contrib == nil {
		t.Fatal("BuildInitConfig() returned nil, want non-nil for enabled state")
	}
	if contrib.PluginKey() != "cost" {
		t.Errorf("PluginKey() = %q, want %q", contrib.PluginKey(), "cost")
	}
	var cfg model.CostConfig
	if err := contrib.DecodeConfig(&cfg); err != nil {
		t.Fatalf("DecodeConfig() error = %v", err)
	}
	awsCfg, ok := cfg.Providers["aws"]
	if !ok {
		t.Fatal("Config missing providers.aws")
	}
	if !awsCfg.Enabled {
		t.Errorf("Config providers.aws.enabled = false, want true")
	}
}

func TestPlugin_BuildInitConfig_Disabled(t *testing.T) {
	p := newTestPlugin(t)
	state := initwiz.NewStateMap()
	providerEnabledKey("aws").Set(state, false)

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
	// No key set — providerEnabledKey("aws").Get(state) returns false.

	contrib, err := p.BuildInitConfig(state)
	if err != nil {
		t.Fatalf("BuildInitConfig() error = %v", err)
	}

	if contrib != nil {
		t.Errorf("BuildInitConfig() = %v, want nil for unset state", contrib)
	}
}
