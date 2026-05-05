// Package gitlab provides the GitLab CI plugin for TerraCi.
// It registers a pipeline generator and MR comment service.
package gitlab

import (
	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/pkg/plugin/registry"
	configpkg "github.com/edelwud/terraci/plugins/gitlab/internal/config"
)

// pluginName is the canonical provider name of the GitLab plugin.
const pluginName = "gitlab"

// defaultStagesPrefix is the default GitLab CI stages-prefix used by both
// the plugin defaults and the init wizard.
const defaultStagesPrefix = "deploy"

func init() {
	registry.RegisterFactory(func() plugin.Plugin {
		return &Plugin{BasePlugin: plugin.BasePlugin[*configpkg.Config]{
			PluginName: pluginName,
			PluginDesc: "GitLab CI pipeline generation and MR comments",
			EnableMode: plugin.EnabledWhenConfigured,
			DefaultCfg: func() *configpkg.Config {
				return &configpkg.Config{
					Image:        configpkg.Image{Name: "hashicorp/terraform:1.6"},
					StagesPrefix: defaultStagesPrefix,
					Parallelism:  5,
					CacheEnabled: true,
				}
			},
		}}
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
	}
}

// SetAutoApprove sets auto-approve mode directly on the typed config.
func (p *Plugin) SetAutoApprove(v bool) {
	if cfg := p.Config(); cfg != nil {
		cfg.AutoApprove = v
	}
}
