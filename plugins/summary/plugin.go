// Package summary provides the summary plugin for TerraCi.
// It registers a pipeline contributor (PhaseFinalize) and the `terraci summary` command.
package summary

import (
	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/pkg/plugin/registry"
	summaryengine "github.com/edelwud/terraci/plugins/summary/internal"
)

func init() {
	registry.Register(&Plugin{
		BasePlugin: plugin.BasePlugin[*summaryengine.Config]{
			PluginName: "summary",
			PluginDesc: "MR/PR comment posting from plan results",
			EnableMode: plugin.EnabledByDefault,
			DefaultCfg: func() *summaryengine.Config {
				return &summaryengine.Config{}
			},
			IsEnabledFn: func(cfg *summaryengine.Config) bool {
				return cfg == nil || cfg.Enabled == nil || *cfg.Enabled
			},
		},
	})
}

// Plugin is the summary plugin.
type Plugin struct {
	plugin.BasePlugin[*summaryengine.Config]
}
