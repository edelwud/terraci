// Package update provides the Terraform dependency version checker and updater plugin for TerraCi.
package update

import (
	"github.com/edelwud/terraci/pkg/plugin"
	updateengine "github.com/edelwud/terraci/plugins/update/internal"
)

func init() {
	plugin.Register(&Plugin{
		BasePlugin: plugin.BasePlugin[*updateengine.UpdateConfig]{
			PluginName: "update",
			PluginDesc: "Terraform dependency version checker and updater",
			EnableMode: plugin.EnabledExplicitly,
			DefaultCfg: func() *updateengine.UpdateConfig {
				return &updateengine.UpdateConfig{
					Target: updateengine.TargetAll,
					Bump:   updateengine.BumpMinor,
				}
			},
			IsEnabledFn: func(cfg *updateengine.UpdateConfig) bool {
				return cfg != nil && cfg.Enabled
			},
		},
	})
}

// Plugin is the dependency version checker and updater plugin.
type Plugin struct {
	plugin.BasePlugin[*updateengine.UpdateConfig]
	registry      updateengine.RegistryClient
	serviceDirRel string
}

// Reset resets all plugin state.
func (p *Plugin) Reset() {
	p.BasePlugin.Reset()
	p.registry = nil
	p.serviceDirRel = ""
}
