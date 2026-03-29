package summary

import (
	"testing"

	"github.com/edelwud/terraci/pkg/config"
	"github.com/edelwud/terraci/pkg/plugin"
	summaryengine "github.com/edelwud/terraci/plugins/summary/internal"
)

func newTestPlugin() *Plugin {
	return &Plugin{
		BasePlugin: plugin.BasePlugin[*summaryengine.Config]{
			PluginName: "summary",
			PluginDesc: "MR/PR comment posting from plan results",
			EnableMode: plugin.EnabledByDefault,
			DefaultCfg: func() *summaryengine.Config {
				return &summaryengine.Config{}
			},
			IsEnabledFn: func(cfg *summaryengine.Config) bool {
				return cfg == nil || cfg.Enabled == nil || *cfg.Enabled
			},
		},
	}
}

func TestPlugin_BuildInitConfig_Enabled(t *testing.T) {
	p := newTestPlugin()
	state := plugin.NewStateMap()
	state.Set("summary.enabled", true)

	contrib := p.BuildInitConfig(state)
	if contrib == nil {
		t.Fatal("BuildInitConfig() returned nil")
	}
	if got := contrib.Config["enabled"]; got != true {
		t.Fatalf("Config[enabled] = %v, want true", got)
	}
}

func TestPlugin_BuildInitConfig_Disabled(t *testing.T) {
	p := newTestPlugin()
	state := plugin.NewStateMap()
	state.Set("summary.enabled", false)

	contrib := p.BuildInitConfig(state)
	if contrib == nil {
		t.Fatal("BuildInitConfig() returned nil")
	}
	if got := contrib.Config["enabled"]; got != false {
		t.Fatalf("Config[enabled] = %v, want false", got)
	}
}

func TestPlugin_BuildInitConfig_DisabledRoundTripDisablesPlugin(t *testing.T) {
	p := newTestPlugin()
	state := plugin.NewStateMap()
	state.Set("summary.enabled", false)

	contrib := p.BuildInitConfig(state)
	cfg, err := config.BuildConfigFromPlugins("{service}/{environment}/{region}/{module}", map[string]map[string]any{
		contrib.PluginKey: contrib.Config,
	})
	if err != nil {
		t.Fatalf("BuildConfigFromPlugins() error = %v", err)
	}
	if err := p.DecodeAndSet(func(target any) error {
		return cfg.PluginConfig("summary", target)
	}); err != nil {
		t.Fatalf("DecodeAndSet() error = %v", err)
	}
	if p.IsEnabled() {
		t.Fatal("summary plugin should be disabled after round-trip config with enabled=false")
	}
}
