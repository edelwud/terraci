package policy

import (
	"context"
	"testing"

	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/pkg/plugin/plugintest"
	policyconfig "github.com/edelwud/terraci/plugins/policy/internal/config"
)

func newTestPlugin() *Plugin {
	return &Plugin{
		BasePlugin: plugin.BasePlugin[*policyconfig.Config]{
			PluginName: "policy",
			PluginDesc: "OPA policy checks for Terraform plans",
			EnableMode: plugin.EnabledExplicitly,
			DefaultCfg: func() *policyconfig.Config {
				return &policyconfig.Config{}
			},
			IsEnabledFn: func(cfg *policyconfig.Config) bool {
				return cfg != nil && cfg.Enabled
			},
		},
	}
}

func TestPlugin_Preflight_ConfiguredButDisabled(t *testing.T) {
	p := newTestPlugin()
	p.SetTypedConfig(&policyconfig.Config{Enabled: false})
	appCtx := plugintest.NewAppContext(t, t.TempDir())

	if err := p.Preflight(context.Background(), appCtx); err != nil {
		t.Fatalf("Preflight() error = %v", err)
	}
}

func TestPlugin_Runtime_CreatesPuller(t *testing.T) {
	p := newTestPlugin()
	p.SetTypedConfig(&policyconfig.Config{
		Enabled: true,
		Sources: []policyconfig.SourceConfig{{Type: policyconfig.SourceTypePath, Path: "terraform"}},
	})

	runtime := plugintest.MustRuntime[*policyRuntime](t, p, plugintest.NewAppContext(t, t.TempDir()))
	if runtime.sources == nil {
		t.Fatal("runtime.sources should not be nil")
	}
}
