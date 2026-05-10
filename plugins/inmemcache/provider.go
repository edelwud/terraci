package inmemcache

import (
	"context"
	"sync"

	"github.com/edelwud/terraci/pkg/plugin"
)

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
