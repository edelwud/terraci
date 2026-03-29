// Package policy provides the OPA policy check plugin for TerraCi.
package policy

import (
	"github.com/edelwud/terraci/pkg/plugin"
	policyengine "github.com/edelwud/terraci/plugins/policy/internal"
)

func init() { //nolint:gochecknoinits // intentional plugin registration
	plugin.Register(&Plugin{
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
	})
}

// Plugin is the OPA policy check plugin.
type Plugin struct {
	plugin.BasePlugin[*policyengine.Config]
	serviceDirRel string // relative path, for pipeline artifact paths
}

// Reset resets all plugin state.
func (p *Plugin) Reset() {
	p.BasePlugin.Reset()
	p.serviceDirRel = ""
}
