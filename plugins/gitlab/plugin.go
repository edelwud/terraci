// Package gitlab provides the GitLab CI plugin for TerraCi.
// It registers a pipeline generator and MR comment service.
package gitlab

import (
	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/pkg/plugin/registry"
	configpkg "github.com/edelwud/terraci/plugins/gitlab/internal/config"
)

func init() {
	registry.Register(&Plugin{
		BasePlugin: plugin.BasePlugin[*configpkg.Config]{
			PluginName: "gitlab",
			PluginDesc: "GitLab CI pipeline generation and MR comments",
			EnableMode: plugin.EnabledWhenConfigured,
			DefaultCfg: func() *configpkg.Config {
				return &configpkg.Config{
					TerraformBinary: "terraform",
					Image:           configpkg.Image{Name: "hashicorp/terraform:1.6"},
					StagesPrefix:    "deploy",
					Parallelism:     5,
					PlanEnabled:     true,
					InitEnabled:     true,
				}
			},
		},
	})
}

// Plugin is the GitLab CI plugin.
type Plugin struct {
	plugin.BasePlugin[*configpkg.Config]
}

// SetPlanOnly sets plan-only mode directly on the typed config.
func (p *Plugin) SetPlanOnly(v bool) {
	if cfg := p.Config(); cfg != nil {
		cfg.PlanOnly = v
		if v {
			cfg.PlanEnabled = true
		}
	}
}

// SetAutoApprove sets auto-approve mode directly on the typed config.
func (p *Plugin) SetAutoApprove(v bool) {
	if cfg := p.Config(); cfg != nil {
		cfg.AutoApprove = v
	}
}
