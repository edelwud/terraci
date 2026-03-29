package plugin

import (
	"errors"
	"fmt"
	"os"
	"strings"
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

// ciProviderPlugin is a plugin that implements all CI provider interfaces.
// Used internally by ResolveProvider to find qualifying plugins.
type ciProviderPlugin interface {
	Plugin
	EnvDetector
	CIMetadata
	GeneratorFactory
	CommentFactory
}

func isPluginEnabled(p Plugin) bool {
	if cl, ok := p.(ConfigLoader); ok {
		return cl.IsEnabled()
	}
	return true
}

func activeCIProviders() []ciProviderPlugin {
	candidates := ByCapability[ciProviderPlugin]()
	active := make([]ciProviderPlugin, 0, len(candidates))
	for _, c := range candidates {
		if isPluginEnabled(c) {
			active = append(active, c)
		}
	}
	return active
}

// ResolveProvider detects the active CI provider.
// Priority: env detection → TERRACI_PROVIDER env → single registered → configured.
func ResolveProvider() (*CIProvider, error) {
	candidates := activeCIProviders()
	if len(candidates) == 0 {
		return nil, errors.New("no active CI provider plugins registered")
	}

	// Check env detection (CI environment variables)
	for _, c := range candidates {
		if c.DetectEnv() {
			return buildCIProvider(c), nil
		}
	}

	// Check TERRACI_PROVIDER env var
	if name := os.Getenv("TERRACI_PROVIDER"); name != "" {
		return findProvider(candidates, name)
	}

	// Single provider registered
	if len(candidates) == 1 {
		return buildCIProvider(candidates[0]), nil
	}

	return nil, fmt.Errorf("cannot determine CI provider: multiple plugins registered (%s), set TERRACI_PROVIDER", providerNames(candidates))
}

func buildCIProvider(p ciProviderPlugin) *CIProvider {
	return NewCIProvider(p, p, p, p)
}

// ResolveChangeDetector returns the active ChangeDetectionProvider.
// Priority: single registered → configured → first available.
func ResolveChangeDetector() (ChangeDetectionProvider, error) {
	detectors := ByCapability[ChangeDetectionProvider]()
	if len(detectors) == 0 {
		return nil, errors.New("no change detection plugin registered")
	}
	if len(detectors) == 1 {
		return detectors[0], nil
	}
	for _, d := range detectors {
		if cl, ok := d.(ConfigLoader); ok && cl.IsEnabled() {
			return d, nil
		}
	}
	return detectors[0], nil
}

func findProvider(candidates []ciProviderPlugin, name string) (*CIProvider, error) {
	for _, c := range candidates {
		if c.ProviderName() == name {
			return buildCIProvider(c), nil
		}
	}
	return nil, fmt.Errorf("provider %q not found (available: %s)", name, providerNames(candidates))
}

func providerNames(candidates []ciProviderPlugin) string {
	names := ""
	var namesSb141 strings.Builder
	for i, c := range candidates {
		if i > 0 {
			namesSb141.WriteString(", ")
		}
		namesSb141.WriteString(c.ProviderName())
	}
	names += namesSb141.String()
	return names
}

// InitializablesForStartup returns plugins that should participate in lifecycle
// initialization for the current config state.
func InitializablesForStartup() []Initializable {
	initializables := ByCapability[Initializable]()
	result := make([]Initializable, 0, len(initializables))
	for _, p := range initializables {
		if isPluginEnabled(p) {
			result = append(result, p)
		}
	}
	return result
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
