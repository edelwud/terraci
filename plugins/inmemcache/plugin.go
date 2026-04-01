// Package inmemcache provides a built-in process-local KV cache backend.
package inmemcache

import (
	"context"
	"sync"

	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/pkg/plugin/registry"
)

func init() {
	registry.Register(&Plugin{
		BasePlugin: plugin.BasePlugin[*Config]{
			PluginName: "inmemcache",
			PluginDesc: "Built-in process-local KV cache backend",
			EnableMode: plugin.EnabledByDefault,
			DefaultCfg: func() *Config { return &Config{Enabled: true} },
			IsEnabledFn: func(cfg *Config) bool {
				return cfg == nil || cfg.Enabled
			},
		},
		cache: newCache(),
	})
}

// Config controls whether the built-in in-memory cache backend is active.
type Config struct {
	Enabled bool `yaml:"enabled,omitempty" json:"enabled,omitempty" jsonschema:"description=Enable the built-in in-memory KV cache backend,default=true"`
}

// Plugin is the built-in in-memory KV cache backend.
type Plugin struct {
	plugin.BasePlugin[*Config]
	cache *cache
	mu    sync.Mutex
}

// NewKVCache returns the shared in-memory cache backend instance.
func (p *Plugin) NewKVCache(_ context.Context, _ *plugin.AppContext) (plugin.KVCache, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.cache == nil {
		p.cache = newCache()
	}

	return p.cache, nil
}

// Reset resets the cache state for tests.
func (p *Plugin) Reset() {
	p.BasePlugin.Reset()
	p.mu.Lock()
	defer p.mu.Unlock()
	p.cache = newCache()
}
