// Package cost provides the AWS cost estimation plugin for TerraCi.
package cost

import (
	"github.com/edelwud/terraci/pkg/plugin"
	costengine "github.com/edelwud/terraci/plugins/cost/internal"
)

func init() { //nolint:gochecknoinits // intentional plugin registration
	plugin.Register(&Plugin{
		BasePlugin: plugin.BasePlugin[*costengine.CostConfig]{
			PluginName: "cost",
			PluginDesc: "AWS cost estimation from Terraform plans",
			EnableMode: plugin.EnabledExplicitly,
			DefaultCfg: func() *costengine.CostConfig {
				return &costengine.CostConfig{}
			},
			IsEnabledFn: func(cfg *costengine.CostConfig) bool {
				return cfg != nil && cfg.Enabled
			},
		},
	})
}

// Plugin is the AWS cost estimation plugin.
type Plugin struct {
	plugin.BasePlugin[*costengine.CostConfig]
	estimator     *costengine.Estimator
	serviceDirRel string // relative path, for pipeline artifact paths
}

// Reset resets all plugin state.
func (p *Plugin) Reset() {
	p.BasePlugin.Reset()
	p.estimator = nil
	p.serviceDirRel = ""
}

func (p *Plugin) getEstimator() *costengine.Estimator {
	return p.estimator
}
