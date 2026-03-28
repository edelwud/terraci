package plugin

import (
	"fmt"
	"os"
	"sync"
)

var (
	mu      sync.Mutex
	plugins = make(map[string]Plugin)
	order   []string
)

// Register adds a plugin to the global registry. Called from init() in plugin packages.
// Panics on duplicate names (fail-fast at startup).
func Register(p Plugin) {
	mu.Lock()
	defer mu.Unlock()
	if _, exists := plugins[p.Name()]; exists {
		panic("terraci: duplicate plugin: " + p.Name())
	}
	plugins[p.Name()] = p
	order = append(order, p.Name())
}

// All returns registered plugins in registration order.
func All() []Plugin {
	mu.Lock()
	defer mu.Unlock()
	result := make([]Plugin, 0, len(order))
	for _, name := range order {
		result = append(result, plugins[name])
	}
	return result
}

// Get returns a plugin by name.
func Get(name string) (Plugin, bool) {
	mu.Lock()
	defer mu.Unlock()
	p, ok := plugins[name]
	return p, ok
}

// ByCapability returns all plugins that implement the given capability interface.
func ByCapability[T Plugin]() []T {
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

// ResolveProvider detects the active CI provider from registered GeneratorProviders.
// Priority: env detection → TERRACI_PROVIDER env → single registered → default gitlab fallback.
func ResolveProvider() (GeneratorProvider, error) {
	generators := ByCapability[GeneratorProvider]()
	if len(generators) == 0 {
		return nil, fmt.Errorf("no CI provider plugins registered")
	}

	// Check env detection (CI environment variables)
	for _, g := range generators {
		if g.DetectEnv() {
			return g, nil
		}
	}

	// Check TERRACI_PROVIDER env var
	if name := os.Getenv("TERRACI_PROVIDER"); name != "" {
		return findProvider(generators, name)
	}

	// Single provider registered
	if len(generators) == 1 {
		return generators[0], nil
	}

	// Filter by explicitly configured providers
	var configured []GeneratorProvider
	for _, g := range generators {
		if cp, ok := g.(ConfigProvider); ok && cp.IsConfigured() {
			configured = append(configured, g)
		}
	}
	if len(configured) == 1 {
		return configured[0], nil
	}

	return nil, fmt.Errorf("cannot determine CI provider: multiple plugins registered (%s), set TERRACI_PROVIDER", providerNames(generators))
}

// ResolveChangeDetector returns the active ChangeDetectionProvider.
// Priority: single registered → configured → first available.
func ResolveChangeDetector() (ChangeDetectionProvider, error) {
	detectors := ByCapability[ChangeDetectionProvider]()
	if len(detectors) == 0 {
		return nil, fmt.Errorf("no change detection plugin registered")
	}
	if len(detectors) == 1 {
		return detectors[0], nil
	}
	for _, d := range detectors {
		if cp, ok := d.(ConfigProvider); ok && cp.IsConfigured() {
			return d, nil
		}
	}
	return detectors[0], nil
}

func findProvider(generators []GeneratorProvider, name string) (GeneratorProvider, error) {
	for _, g := range generators {
		if g.ProviderName() == name {
			return g, nil
		}
	}
	return nil, fmt.Errorf("provider %q not found (available: %s)", name, providerNames(generators))
}

func providerNames(generators []GeneratorProvider) string {
	names := ""
	for i, g := range generators {
		if i > 0 {
			names += ", "
		}
		names += g.ProviderName()
	}
	return names
}

// Reset clears the registry. Only for testing.
func Reset() {
	mu.Lock()
	defer mu.Unlock()
	plugins = make(map[string]Plugin)
	order = nil
}

// ResetPlugins resets mutable state on all registered plugins that implement Resettable.
// The registry itself is NOT cleared — plugins stay registered, only their internal state
// (config, flags, cached clients) is zeroed. Intended for test isolation.
func ResetPlugins() {
	mu.Lock()
	defer mu.Unlock()
	for _, name := range order {
		if r, ok := plugins[name].(Resettable); ok {
			r.Reset()
		}
	}
}
