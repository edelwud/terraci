// Package summary provides the summary plugin for TerraCi.
// It registers a pipeline contributor and the `terraci summary` command.
package summary

import (
	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/pkg/plugin/registry"
	summaryengine "github.com/edelwud/terraci/plugins/summary/internal/summaryengine"
)

// pluginName is the canonical plugin identifier used in commands, reports and config.
const pluginName = "summary"

func init() {
	registry.RegisterFactory(func() plugin.Plugin {
		return &Plugin{BasePlugin: plugin.BasePlugin[*summaryengine.Config]{
			PluginName: pluginName,
			PluginDesc: "MR/PR comment posting from plan results",
			EnableMode: plugin.EnabledByDefault,
			DefaultCfg: func() *summaryengine.Config {
				return &summaryengine.Config{}
			},
			IsEnabledFn: func(cfg *summaryengine.Config) bool {
				return cfg == nil || cfg.Enabled == nil || *cfg.Enabled
			},
		}}
	})
}

// Plugin is the summary plugin.
type Plugin struct {
	plugin.BasePlugin[*summaryengine.Config]
}
