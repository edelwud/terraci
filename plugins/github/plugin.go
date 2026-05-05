// Package github provides the GitHub Actions plugin for TerraCi.
// It registers a pipeline generator and PR comment service.
package github

import (
	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/pkg/plugin/registry"
	configpkg "github.com/edelwud/terraci/plugins/github/internal/config"
)

// pluginName is the canonical provider name of the GitHub Actions plugin.
const pluginName = "github"

func init() {
	registry.RegisterFactory(func() plugin.Plugin {
		return &Plugin{
			BasePlugin: newBasePlugin(),
		}
	})
}

func newBasePlugin() plugin.BasePlugin[*configpkg.Config] {
	return plugin.BasePlugin[*configpkg.Config]{
		PluginName: pluginName,
		PluginDesc: "GitHub Actions pipeline generation and PR comments",
		EnableMode: plugin.EnabledWhenConfigured,
		DefaultCfg: func() *configpkg.Config {
			return &configpkg.Config{
				RunsOn: "ubuntu-latest",
			}
		},
	}
}

// Plugin is the GitHub Actions plugin.
type Plugin struct {
	plugin.BasePlugin[*configpkg.Config]
}

// SetPlanOnly sets plan-only mode directly on the typed config.
func (p *Plugin) SetPlanOnly(v bool) {
	if cfg := p.Config(); cfg != nil {
		cfg.PlanOnly = v
	}
}

// SetAutoApprove sets auto-approve mode directly on the typed config.
func (p *Plugin) SetAutoApprove(v bool) {
	if cfg := p.Config(); cfg != nil {
		cfg.AutoApprove = v
	}
}
