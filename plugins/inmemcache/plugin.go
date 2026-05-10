// Package inmemcache provides a built-in process-local KV cache backend.
package inmemcache

import (
	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/pkg/plugin/registry"
)

func init() {
	registry.RegisterFactory(func() plugin.Plugin {
		return &Plugin{BasePlugin: plugin.BasePlugin[*Config]{
			PluginName: "inmemcache",
			PluginDesc: "Built-in process-local KV cache backend",
			EnableMode: plugin.EnabledByDefault,
			DefaultCfg: func() *Config { return &Config{Enabled: true} },
			IsEnabledFn: func(cfg *Config) bool {
				return cfg == nil || cfg.Enabled
			},
		},
			cache: newCache(),
		}
	})
}
