// Package cost provides the cloud cost estimation plugin for TerraCi.
package cost

import (
	// Register the built-in AWS provider with the cost engine.
	_ "github.com/edelwud/terraci/plugins/cost/internal/cloud/aws"

	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/pkg/plugin/registry"
	"github.com/edelwud/terraci/plugins/cost/internal/model"
)

// pluginName is the canonical plugin identifier used in commands, reports and config.
const pluginName = "cost"

func init() {
	registry.Register(&Plugin{
		BasePlugin: plugin.BasePlugin[*model.CostConfig]{
			PluginName: pluginName,
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
