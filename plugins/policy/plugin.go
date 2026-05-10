// Package policy provides the OPA policy check plugin for TerraCi.
package policy

import (
	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/pkg/plugin/registry"
	policyconfig "github.com/edelwud/terraci/plugins/policy/internal/config"
)

// pluginName is the canonical plugin identifier used in commands, reports and config.
const pluginName = "policy"

func init() {
	registry.RegisterFactory(func() plugin.Plugin {
		return &Plugin{BasePlugin: plugin.BasePlugin[*policyconfig.Config]{
			PluginName: pluginName,
			PluginDesc: "OPA policy checks for Terraform plans",
			EnableMode: plugin.EnabledExplicitly,
			DefaultCfg: func() *policyconfig.Config {
				return &policyconfig.Config{}
			},
			IsEnabledFn: func(cfg *policyconfig.Config) bool {
				return cfg != nil && cfg.Enabled
			},
		}}
	})
}

// Plugin is the OPA policy check plugin.
type Plugin struct {
	plugin.BasePlugin[*policyconfig.Config]
}
