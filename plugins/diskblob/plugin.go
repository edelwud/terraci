// Package diskblob provides a filesystem-backed blob store backend.
package diskblob

import (
	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/pkg/plugin/registry"
)

func init() {
	registry.RegisterFactory(func() plugin.Plugin {
		return &Plugin{BasePlugin: plugin.BasePlugin[*Config]{
			PluginName: "diskblob",
			PluginDesc: "Filesystem-backed blob store backend",
			EnableMode: plugin.EnabledByDefault,
			DefaultCfg: func() *Config { return &Config{Enabled: true} },
			IsEnabledFn: func(cfg *Config) bool {
				return cfg == nil || cfg.Enabled
			},
		}}
	})
}

// Plugin is the filesystem-backed blob store backend.
type Plugin struct {
	plugin.BasePlugin[*Config]
}
