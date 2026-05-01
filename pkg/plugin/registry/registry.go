// Package registry provides TerraCi's plugin catalog and per-run plugin sets.
// Plugin packages register factories via init(); commands instantiate a fresh
// Registry from those factories for each app run. The Registry implements the
// plugin.Resolver contract and is what plugins receive through AppContext.
package registry

import (
	"sync"

	"github.com/edelwud/terraci/pkg/plugin"
)

// Factory constructs a fresh plugin instance.
type Factory func() plugin.Plugin

type descriptor struct {
	factory Factory
}

// Catalog stores plugin factories and creates isolated per-command plugin sets.
type Catalog struct {
	mu          sync.Mutex
	descriptors map[string]descriptor
	order       []string
}

// NewCatalog creates an empty plugin factory catalog.
func NewCatalog() *Catalog {
	return &Catalog{descriptors: make(map[string]descriptor)}
}

var defaultCatalog = NewCatalog()

// RegisterFactory adds a plugin factory to the global catalog. Called from
// init() in plugin packages. Panics on duplicate names (fail-fast at startup).
func RegisterFactory(factory Factory) {
	defaultCatalog.RegisterFactory(factory)
}

// RegisterFactory adds a plugin factory to this catalog.
func (c *Catalog) RegisterFactory(factory Factory) {
	if factory == nil {
		panic("terraci: nil plugin factory")
	}

	prototype := factory()
	if prototype == nil {
		panic("terraci: nil plugin from factory")
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if _, exists := c.descriptors[prototype.Name()]; exists {
		panic("terraci: duplicate plugin: " + prototype.Name())
	}
	c.descriptors[prototype.Name()] = descriptor{factory: factory}
	c.order = append(c.order, prototype.Name())
}

// Registry is an isolated plugin instance set for one app run. It implements
// plugin.Resolver — both as the lookup surface for capability discovery and as
// the policy resolver for canonical capabilities (CI provider, change detector,
// cache backends, pipeline contributions, preflights).
type Registry struct {
	plugins map[string]plugin.Plugin
	order   []string
}

// New instantiates a fresh plugin set from the registered global factories.
func New() *Registry {
	return defaultCatalog.NewRegistry()
}

// NewRegistry instantiates a fresh plugin set from this catalog.
func (c *Catalog) NewRegistry() *Registry {
	c.mu.Lock()
	defer c.mu.Unlock()
	r := &Registry{
		plugins: make(map[string]plugin.Plugin, len(c.order)),
		order:   append([]string(nil), c.order...),
	}
	for _, name := range c.order {
		p := c.descriptors[name].factory()
		if p == nil {
			panic("terraci: nil plugin from factory: " + name)
		}
		r.plugins[name] = p
	}
	return r
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

// GetPlugin returns a plugin by name.
func (r *Registry) GetPlugin(name string) (plugin.Plugin, bool) {
	if r == nil {
		return nil, false
	}
	p, ok := r.plugins[name]
	return p, ok
}

// ByCapabilityFrom returns all plugins from resolver that implement the given
// capability interface.
func ByCapabilityFrom[T plugin.Plugin](resolver plugin.Resolver) []T {
	if resolver == nil {
		return nil
	}
	var result []T
	for _, p := range resolver.All() {
		if t, ok := p.(T); ok {
			result = append(result, t)
		}
	}
	return result
}

// Reset clears the global catalog. Only for testing.
func Reset() {
	defaultCatalog.Reset()
}

// Reset clears the catalog. Only for testing.
func (c *Catalog) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.descriptors = make(map[string]descriptor)
	c.order = nil
}

func isPluginEnabled(p plugin.Plugin) bool {
	if cl, ok := p.(plugin.ConfigLoader); ok {
		return cl.IsEnabled()
	}
	return true
}
