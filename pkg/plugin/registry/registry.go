// Package registry provides TerraCi's plugin catalog and per-run plugin sets.
// Plugin packages register factories via init(); commands instantiate a fresh
// Registry from those factories for each app run.
package registry

import (
	"sync"

	"github.com/edelwud/terraci/pkg/plugin"
)

type Factory func() plugin.Plugin

type descriptor struct {
	factory Factory
}

var (
	mu          sync.Mutex
	descriptors = make(map[string]descriptor)
	order       []string
	defaultSet  *Registry
)

// Register adds a prebuilt plugin to the global catalog.
//
// Production plugins should prefer RegisterFactory so each app run receives
// fresh plugin instances. Register remains useful for focused tests that want
// to install a specific in-memory double.
func Register(p plugin.Plugin) {
	RegisterFactory(func() plugin.Plugin { return p })
}

// RegisterFactory adds a plugin factory to the global catalog. Called from
// init() in plugin packages.
// Panics on duplicate names (fail-fast at startup).
func RegisterFactory(factory Factory) {
	if factory == nil {
		panic("terraci: nil plugin factory")
	}

	prototype := factory()
	if prototype == nil {
		panic("terraci: nil plugin from factory")
	}

	mu.Lock()
	defer mu.Unlock()
	if _, exists := descriptors[prototype.Name()]; exists {
		panic("terraci: duplicate plugin: " + prototype.Name())
	}
	descriptors[prototype.Name()] = descriptor{factory: factory}
	order = append(order, prototype.Name())
	defaultSet = nil
}

// Registry is an isolated plugin instance set for one app run.
type Registry struct {
	plugins map[string]plugin.Plugin
	order   []string
}

// New instantiates a fresh plugin set from the registered factories.
func New() *Registry {
	mu.Lock()
	defer mu.Unlock()
	return instantiateLocked()
}

// Default returns the package-level plugin set used by legacy package
// functions. Runtime code should prefer an app-owned Registry from New().
func Default() *Registry {
	mu.Lock()
	defer mu.Unlock()
	if defaultSet == nil {
		defaultSet = instantiateLocked()
	}
	return defaultSet
}

func instantiateLocked() *Registry {
	r := &Registry{
		plugins: make(map[string]plugin.Plugin, len(order)),
		order:   append([]string(nil), order...),
	}
	for _, name := range order {
		p := descriptors[name].factory()
		if p == nil {
			panic("terraci: nil plugin from factory: " + name)
		}
		if p.Name() != name {
			panic("terraci: plugin factory name changed: " + name + " -> " + p.Name())
		}
		r.plugins[name] = p
	}
	return r
}

// All returns registered plugins in registration order.
func All() []plugin.Plugin {
	return Default().All()
}

// All returns plugins in registration order.
func (r *Registry) All() []plugin.Plugin {
	if r == nil {
		return nil
	}
	result := make([]plugin.Plugin, 0, len(r.order))
	for _, name := range r.order {
		result = append(result, r.plugins[name])
	}
	return result
}

// Get returns a plugin by name.
func Get(name string) (plugin.Plugin, bool) {
	return Default().Get(name)
}

// Get returns a plugin by name.
func (r *Registry) Get(name string) (plugin.Plugin, bool) {
	if r == nil {
		return nil, false
	}
	p, ok := r.plugins[name]
	return p, ok
}

// ByCapability returns all plugins that implement the given capability interface.
func ByCapability[T plugin.Plugin]() []T {
	return ByCapabilityFrom[T](Default())
}

// ByCapabilityFrom returns all plugins in r that implement the given capability interface.
func ByCapabilityFrom[T plugin.Plugin](r *Registry) []T {
	if r == nil {
		return nil
	}
	var result []T
	for _, name := range r.order {
		if t, ok := r.plugins[name].(T); ok {
			result = append(result, t)
		}
	}
	return result
}

// Reset clears the registry. Only for testing.
func Reset() {
	mu.Lock()
	defer mu.Unlock()
	descriptors = make(map[string]descriptor)
	order = nil
	defaultSet = nil
}

// ResetPlugins resets mutable state on all registered plugins that implement Resettable.
// The registry itself is NOT cleared — plugins stay registered, only their internal state
// (config, flags, cached clients) is zeroed. Intended for test isolation.
func ResetPlugins() {
	for _, p := range Default().All() {
		if r, ok := p.(plugin.Resettable); ok {
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
