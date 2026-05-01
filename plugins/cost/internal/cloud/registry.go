// Package cloud provides the cloud provider registry for cost estimation.
// Cloud providers register themselves via init() using Register.
package cloud

import (
	"sync"

	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcedef"
)

// ResourceRegistration binds a supported Terraform resource type to its runtime definition.
type ResourceRegistration struct {
	Type       resourcedef.ResourceType
	Definition resourcedef.Definition
}

// Definition is the provider-neutral runtime contract for a cloud provider.
type Definition struct {
	// ConfigKey is the YAML key under extensions.cost.providers that enables this provider.
	// Example: "aws" maps to `extensions.cost.providers.aws.enabled: true`.
	// It must match the key used in CostProvidersConfig.
	ConfigKey      string
	Manifest       pricing.ProviderManifest
	FetcherFactory func() pricing.PriceFetcher
	Resources      []ResourceRegistration
}

// Provider encapsulates everything a cloud vendor contributes to the estimator.
// Implement this interface to add support for a new cloud provider (e.g., GCP, Azure).
//
// Registration pattern (same as terraci plugin system):
//
//	func init() {
//	    cloud.Register(&myProvider{})
//	}
type Provider interface {
	// Definition returns the provider-neutral runtime contract owned by this provider.
	Definition() Definition
}

var (
	cpMu      sync.RWMutex
	providers = make(map[string]Provider)
	cpOrder   []string
)

// Register adds a cloud provider to the global registry.
// Called from init() in provider files. Panics on duplicate names.
func Register(cp Provider) {
	cpMu.Lock()
	defer cpMu.Unlock()
	def := cp.Definition()
	if _, exists := providers[def.Manifest.ID]; exists {
		panic("cloud: duplicate cloud provider: " + def.Manifest.ID)
	}
	providers[def.Manifest.ID] = cp
	cpOrder = append(cpOrder, def.Manifest.ID)
}

// Providers returns all registered cloud providers in registration order.
func Providers() []Provider {
	cpMu.RLock()
	defer cpMu.RUnlock()
	result := make([]Provider, 0, len(cpOrder))
	for _, name := range cpOrder {
		result = append(result, providers[name])
	}
	return result
}

// Get returns a registered cloud provider by its provider ID.
func Get(name string) (Provider, bool) {
	cpMu.RLock()
	defer cpMu.RUnlock()
	cp, ok := providers[name]
	return cp, ok
}
