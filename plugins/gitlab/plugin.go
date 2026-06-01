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
				cacheEnabled := true
				return &configpkg.Config{
					StagesPrefix: defaultStagesPrefix,
					Cache:        &configpkg.CacheConfig{Enabled: &cacheEnabled},
				}
			},
		}}
	})
}

// Plugin is the GitLab CI plugin.
type Plugin struct {
	plugin.BasePlugin[*configpkg.Config]
}
