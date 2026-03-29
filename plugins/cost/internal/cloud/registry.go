// Package cloud provides the cloud provider registry for cost estimation.
// Cloud providers register themselves via init() using Register.
package cloud

import (
	"sync"

	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
)

// Provider encapsulates everything a cloud vendor contributes to the estimator.
// Implement this interface to add support for a new cloud provider (e.g., GCP, Azure).
//
// Registration pattern (same as terraci plugin system):
//
//	func init() {
//	    cloud.Register(&myProvider{})
//	}
type Provider interface {
	// Name returns the provider identifier (e.g., "aws", "gcp", "azure").
	Name() string
	// NewFetcher creates a PriceFetcher for this provider's pricing API.
	NewFetcher() pricing.PriceFetcher
	// RegisterHandlers populates the given registry with this provider's resource handlers.
	RegisterHandlers(r *handler.Registry)
	// InitRegionMapping sets up region code to pricing name mappings.
	InitRegionMapping()
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
	if _, exists := providers[cp.Name()]; exists {
		panic("cloud: duplicate cloud provider: " + cp.Name())
	}
	providers[cp.Name()] = cp
	cpOrder = append(cpOrder, cp.Name())
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
