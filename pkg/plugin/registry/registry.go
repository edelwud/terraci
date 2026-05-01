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
)

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

// NewFromFactories creates an isolated registry from explicit factories without
// touching the process-wide plugin catalog.
func NewFromFactories(factories ...Factory) *Registry {
	r := &Registry{
		plugins: make(map[string]plugin.Plugin, len(factories)),
		order:   make([]string, 0, len(factories)),
	}
	for _, factory := range factories {
		if factory == nil {
			panic("terraci: nil plugin factory")
		}
		p := factory()
		if p == nil {
			panic("terraci: nil plugin from factory")
		}
		name := p.Name()
		if _, exists := r.plugins[name]; exists {
			panic("terraci: duplicate plugin: " + name)
		}
		r.plugins[name] = p
		r.order = append(r.order, name)
	}
	return r
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
func (r *Registry) Get(name string) (plugin.Plugin, bool) {
	return r.GetPlugin(name)
}

// GetPlugin returns a plugin by name.
func (r *Registry) GetPlugin(name string) (plugin.Plugin, bool) {
	if r == nil {
		return nil, false
	}
	p, ok := r.plugins[name]
	return p, ok
}

// ByCapabilityFrom returns all plugins in source that implement the given
// capability interface.
func ByCapabilityFrom[T plugin.Plugin](source plugin.Source) []T {
	if source == nil {
		return nil
	}
	var result []T
	for _, p := range source.All() {
		if t, ok := p.(T); ok {
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
}

func isPluginEnabled(p plugin.Plugin) bool {
	if cl, ok := p.(plugin.ConfigLoader); ok {
		return cl.IsEnabled()
	}
	return true
}
