package summary

import (
	"testing"

	"github.com/edelwud/terraci/pkg/config"
	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/pkg/plugin/initwiz"
	summaryengine "github.com/edelwud/terraci/plugins/summary/internal/summaryengine"
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
	state := initwiz.NewStateMap()
	state.Set("summary.enabled", true)

	contrib, err := p.BuildInitConfig(state)
	if err != nil {
		t.Fatalf("BuildInitConfig() error = %v", err)
	}
	if contrib != nil {
		t.Fatalf("BuildInitConfig() = %#v, want nil for default-enabled summary", contrib)
	}
}

func TestPlugin_BuildInitConfig_Disabled(t *testing.T) {
	p := newTestPlugin()
	state := initwiz.NewStateMap()
	state.Set("summary.enabled", false)

	contrib, err := p.BuildInitConfig(state)
	if err != nil {
		t.Fatalf("BuildInitConfig() error = %v", err)
	}
	if contrib == nil {
		t.Fatal("BuildInitConfig() returned nil")
	}
	var summaryCfg summaryengine.Config
	if err := contrib.DecodeConfig(&summaryCfg); err != nil {
		t.Fatalf("DecodeConfig() error = %v", err)
	}
	if summaryCfg.Enabled == nil || *summaryCfg.Enabled {
		t.Fatalf("Config.Enabled = %#v, want false", summaryCfg.Enabled)
	}
}

func TestPlugin_BuildInitConfig_DisabledRoundTripDisablesPlugin(t *testing.T) {
	p := newTestPlugin()
	state := initwiz.NewStateMap()
	state.Set("summary.enabled", false)

	contrib, err := p.BuildInitConfig(state)
	if err != nil {
		t.Fatalf("BuildInitConfig() error = %v", err)
	}
	execution := config.DefaultConfig().Execution
	execution.Binary = "terraform"
	execution.PlanEnabled = true
	execution.InitEnabled = true
	extensions, err := config.NewExtensionSet(contrib.ExtensionValue())
	if err != nil {
		t.Fatalf("NewExtensionSet() error = %v", err)
	}
	cfg, err := config.Build(config.BuildOptions{
		Pattern:    "{service}/{environment}/{region}/{module}",
		Execution:  &execution,
		Extensions: extensions,
	})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if err := p.DecodeAndSet(func(target any) error {
		return cfg.Extension("summary", target)
	}); err != nil {
		t.Fatalf("DecodeAndSet() error = %v", err)
	}
	if p.IsEnabled() {
		t.Fatal("summary plugin should be disabled after round-trip config with enabled=false")
	}
}
