// Package github provides the GitHub Actions plugin for TerraCi.
// It registers a pipeline generator and PR comment service.
package github

import (
	"github.com/edelwud/terraci/pkg/plugin"
	githubci "github.com/edelwud/terraci/plugins/github/internal"
)

func init() {
	plugin.Register(&Plugin{
		BasePlugin: plugin.BasePlugin[*githubci.Config]{
			PluginName: "github",
			PluginDesc: "GitHub Actions pipeline generation and PR comments",
			EnableMode: plugin.EnabledWhenConfigured,
			DefaultCfg: func() *githubci.Config {
				return &githubci.Config{
					TerraformBinary: "terraform",
					RunsOn:          "ubuntu-latest",
					PlanEnabled:     true,
					InitEnabled:     true,
				}
			},
		},
	})
}

// Plugin is the GitHub Actions plugin.
type Plugin struct {
	plugin.BasePlugin[*githubci.Config]
}

// Reset resets all plugin state.
func (p *Plugin) Reset() {
	p.BasePlugin.Reset()
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
