// Package cost provides the cloud cost estimation plugin for TerraCi.
package cost

import (
	// Register the built-in AWS provider with the cost engine.
	_ "github.com/edelwud/terraci/plugins/cost/internal/cloud/aws"

	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/plugins/cost/internal/model"
)

func init() {
	plugin.Register(&Plugin{
		BasePlugin: plugin.BasePlugin[*model.CostConfig]{
			PluginName: "cost",
			PluginDesc: "Cloud cost estimation from Terraform plans",
			EnableMode: plugin.EnabledExplicitly,
			DefaultCfg: func() *model.CostConfig {
				return &model.CostConfig{}
			},
			IsEnabledFn: func(cfg *model.CostConfig) bool {
				return cfg != nil && cfg.HasEnabledProviders()
			},
		},
	})
}

// Plugin is the cloud cost estimation plugin.
type Plugin struct {
	plugin.BasePlugin[*model.CostConfig]
}

// Reset resets all plugin state.
func (p *Plugin) Reset() {
	p.BasePlugin.Reset()
}
