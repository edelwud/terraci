// Package git provides the Git change detection plugin for TerraCi.
package git

import (
	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/pkg/plugin/registry"
)

const pluginName = "git"

func init() {
	registry.RegisterFactory(func() plugin.Plugin {
		return &Plugin{BasePlugin: plugin.BasePlugin[*Config]{
			PluginName: pluginName,
			PluginDesc: "Git change detection for incremental pipelines",
			EnableMode: plugin.EnabledAlways,
			DefaultCfg: func() *Config { return &Config{} },
		}}
	})
}

// Plugin is the Git change detection plugin.
type Plugin struct {
	plugin.BasePlugin[*Config]
}
