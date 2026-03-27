// Package cost provides the AWS cost estimation plugin for TerraCi.
package cost

import (
	"github.com/edelwud/terraci/pkg/plugin"
	costengine "github.com/edelwud/terraci/plugins/cost/internal"
)

func init() { //nolint:gochecknoinits // intentional plugin registration
	plugin.Register(&Plugin{})
}

// Plugin is the AWS cost estimation plugin.
type Plugin struct {
	cfg           *costengine.CostConfig
	estimator     *costengine.Estimator
	configured    bool
	serviceDirRel string // relative path, for pipeline artifact paths
}

func (p *Plugin) Name() string        { return "cost" }
func (p *Plugin) Description() string { return "AWS cost estimation from Terraform plans" }
