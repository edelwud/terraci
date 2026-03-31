// Package registry provides the global plugin registry for TerraCi.
// Plugins register themselves via init() and are discovered by capability
// using the ByCapability generic function.
package registry

import (
	"sync"

	"github.com/edelwud/terraci/pkg/plugin"
)

var (
	mu      sync.Mutex
	plugins = make(map[string]plugin.Plugin)
	order   []string
)

// Register adds a plugin to the global registry. Called from init() in plugin packages.
// Panics on duplicate names (fail-fast at startup).
func Register(p plugin.Plugin) {
	mu.Lock()
	defer mu.Unlock()
	if _, exists := plugins[p.Name()]; exists {
		panic("terraci: duplicate plugin: " + p.Name())
	}
	plugins[p.Name()] = p
	order = append(order, p.Name())
}

// All returns registered plugins in registration order.
func All() []plugin.Plugin {
	mu.Lock()
	defer mu.Unlock()
	result := make([]plugin.Plugin, 0, len(order))
	for _, name := range order {
		result = append(result, plugins[name])
	}
	return result
}

// Get returns a plugin by name.
func Get(name string) (plugin.Plugin, bool) {
	mu.Lock()
	defer mu.Unlock()
	p, ok := plugins[name]
	return p, ok
}

// ByCapability returns all plugins that implement the given capability interface.
func ByCapability[T plugin.Plugin]() []T {
	mu.Lock()
	defer mu.Unlock()
	var result []T
	for _, name := range order {
		if t, ok := plugins[name].(T); ok {
			result = append(result, t)
		}
	}
	return result
}

// Reset clears the registry. Only for testing.
func Reset() {
	mu.Lock()
	defer mu.Unlock()
	plugins = make(map[string]plugin.Plugin)
	order = nil
}

// ResetPlugins resets mutable state on all registered plugins that implement Resettable.
// The registry itself is NOT cleared — plugins stay registered, only their internal state
// (config, flags, cached clients) is zeroed. Intended for test isolation.
func ResetPlugins() {
	mu.Lock()
	defer mu.Unlock()
	for _, name := range order {
		if r, ok := plugins[name].(plugin.Resettable); ok {
			r.Reset()
		}
	}
}

func isPluginEnabled(p plugin.Plugin) bool {
	if cl, ok := p.(plugin.ConfigLoader); ok {
		return cl.IsEnabled()
	}
	return true
}
