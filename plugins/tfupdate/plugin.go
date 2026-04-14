// Package tfupdate provides the Terraform dependency resolver and lock sync plugin for TerraCi.
package tfupdate

import (
	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/pkg/plugin/registry"
	tfupdateengine "github.com/edelwud/terraci/plugins/tfupdate/internal"
)

func init() {
	registry.Register(&Plugin{
		BasePlugin: plugin.BasePlugin[*tfupdateengine.UpdateConfig]{
			PluginName: "tfupdate",
			PluginDesc: "Terraform dependency resolver and lock synchronizer",
			EnableMode: plugin.EnabledExplicitly,
			DefaultCfg: tfupdateengine.DefaultConfig,
			IsEnabledFn: func(cfg *tfupdateengine.UpdateConfig) bool {
				return cfg != nil && cfg.Enabled
			},
		},
	})
}

// Plugin is the Terraform dependency resolver plugin.
type Plugin struct {
	plugin.BasePlugin[*tfupdateengine.UpdateConfig]
	registryFactory func() tfupdateengine.RegistryClient
}

// Reset resets all plugin state.
func (p *Plugin) Reset() {
	p.BasePlugin.Reset()
	p.registryFactory = nil
}
