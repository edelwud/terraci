// Package policy provides the OPA policy check plugin for TerraCi.
package policy

import (
	"github.com/edelwud/terraci/pkg/plugin"
	policyengine "github.com/edelwud/terraci/plugins/policy/internal"
)

func init() { //nolint:gochecknoinits // intentional plugin registration
	plugin.Register(&Plugin{})
}

// Plugin is the OPA policy check plugin.
type Plugin struct {
	cfg           *policyengine.Config
	configured    bool
	serviceDirRel string // relative path, for pipeline artifact paths
}

func (p *Plugin) Name() string        { return "policy" }
func (p *Plugin) Description() string { return "OPA policy checks for Terraform plans" }
