package policy

import (
	"context"
	"testing"

	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/pkg/plugin/plugintest"
	policyengine "github.com/edelwud/terraci/plugins/policy/internal"
)

func newTestPlugin() *Plugin {
	return &Plugin{
		BasePlugin: plugin.BasePlugin[*policyengine.Config]{
			PluginName: "policy",
			PluginDesc: "OPA policy checks for Terraform plans",
			EnableMode: plugin.EnabledExplicitly,
			DefaultCfg: func() *policyengine.Config {
				return &policyengine.Config{}
			},
			IsEnabledFn: func(cfg *policyengine.Config) bool {
				return cfg != nil && cfg.Enabled
			},
		},
	}
}

func TestPlugin_Initialize_ConfiguredButDisabled(t *testing.T) {
	p := newTestPlugin()
	p.SetTypedConfig(&policyengine.Config{Enabled: false})
	appCtx := plugintest.NewAppContext(t, t.TempDir())

	if err := p.Initialize(context.Background(), appCtx); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}
}
