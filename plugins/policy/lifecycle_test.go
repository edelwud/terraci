package policy

import (
	"context"
	"testing"

	"github.com/edelwud/terraci/pkg/config"
	"github.com/edelwud/terraci/pkg/plugin"
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

func newTestAppContext(workDir string) *plugin.AppContext {
	cfg := config.DefaultConfig()
	cfg.ServiceDir = ".terraci"
	return plugin.NewAppContext(cfg, workDir, workDir+"/.terraci", "", plugin.NewReportRegistry())
}

func TestPlugin_Initialize_ConfiguredButDisabled(t *testing.T) {
	p := newTestPlugin()
	p.SetTypedConfig(&policyengine.Config{Enabled: false})
	appCtx := newTestAppContext(t.TempDir())

	if err := p.Initialize(context.Background(), appCtx); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}
}
