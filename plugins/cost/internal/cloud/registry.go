// Package cloud provides the cloud provider registry for cost estimation.
// Cloud providers register themselves via init() using Register.
package cloud

import (
	"sync"

	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
)

// ResourceRegistration binds a supported Terraform resource type to its handler.
type ResourceRegistration struct {
	Type    string
	Handler handler.ResourceHandler
}

// Definition is the provider-neutral runtime contract for a cloud provider.
type Definition struct {
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

// RegisterDefinitionHandlers populates the handler registry from a provider definition.
func RegisterDefinitionHandlers(r *handler.Registry, def Definition) {
	for _, resource := range def.Resources {
		r.Register(def.Manifest.ID, resource.Type, resource.Handler)
	}
}

var (
	cpMu      sync.Mutex
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
	cpMu.Lock()
	defer cpMu.Unlock()
	result := make([]Provider, 0, len(cpOrder))
	for _, name := range cpOrder {
		result = append(result, providers[name])
	}
	return result
}

// Get returns a cloud provider by name.
func Get(name string) (Provider, bool) {
	cpMu.Lock()
	defer cpMu.Unlock()
	cp, ok := providers[name]
	return cp, ok
}

// Reset clears the registry. Only for testing.
func Reset() {
	cpMu.Lock()
	defer cpMu.Unlock()
	providers = make(map[string]Provider)
	cpOrder = nil
}
