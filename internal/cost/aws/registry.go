// Package aws provides AWS resource cost estimation handlers
package aws

import (
	"github.com/caarlos0/log"

	"github.com/edelwud/terraci/internal/cost/pricing"
)

// CostCategory classifies how a handler calculates costs.
type CostCategory int

const (
	// CostCategoryStandard requires AWS Pricing API lookup.
	CostCategoryStandard CostCategory = iota
	// CostCategoryFixed uses hardcoded costs (no API call needed).
	CostCategoryFixed
	// CostCategoryUsageBased is usage-based pricing (returns $0 for fixed estimates).
	CostCategoryUsageBased
)

// ResourceHandler extracts pricing information from terraform resource attributes
type ResourceHandler interface {
	// Category returns how this handler calculates costs.
	Category() CostCategory
	// ServiceCode returns the AWS service code for pricing API
	ServiceCode() pricing.ServiceCode
	// BuildLookup creates a PriceLookup from terraform resource attributes.
	// Not called for Fixed or UsageBased handlers.
	BuildLookup(region string, attrs map[string]any) (*pricing.PriceLookup, error)
	// CalculateCost calculates monthly cost from price and resource attributes.
	// For Fixed handlers, price may be nil.
	CalculateCost(price *pricing.Price, attrs map[string]any) (hourly, monthly float64)
}

// Registry maps terraform resource types to handlers
type Registry struct {
	handlers map[string]ResourceHandler
}

// NewRegistry creates a new resource registry with all supported handlers.
// RegisterAll must be set before calling NewRegistry.
func NewRegistry() *Registry {
	r := &Registry{
		handlers: make(map[string]ResourceHandler),
	}
	if RegisterAll != nil {
		RegisterAll(r)
	}
	return r
}

// RegisterAll registers all built-in resource handlers from subpackages.
// Called by the cost package to avoid import cycles (aws/ ← aws/ec2/ → aws/).
var RegisterAll func(r *Registry)

// Register adds a handler for a resource type
func (r *Registry) Register(resourceType string, handler ResourceHandler) {
	r.handlers[resourceType] = handler
}

// GetHandler returns a handler for a resource type
func (r *Registry) GetHandler(resourceType string) (ResourceHandler, bool) {
	h, ok := r.handlers[resourceType]
	return h, ok
}

// IsSupported checks if a resource type is supported
func (r *Registry) IsSupported(resourceType string) bool {
	_, ok := r.handlers[resourceType]
	return ok
}

// SupportedTypes returns all supported resource types
func (r *Registry) SupportedTypes() []string {
	types := make([]string, 0, len(r.handlers))
	for t := range r.handlers {
		types = append(types, t)
	}
	return types
}

// RequiredServices returns services needed for given resource types
func (r *Registry) RequiredServices(resourceTypes []string) map[pricing.ServiceCode]bool {
	services := make(map[pricing.ServiceCode]bool)
	for _, rt := range resourceTypes {
		if h, ok := r.handlers[rt]; ok {
			services[h.ServiceCode()] = true
		}
	}
	return services
}

// LogUnsupported logs unsupported resource types at debug level
func LogUnsupported(resourceType, address string) {
	log.WithField("type", resourceType).
		WithField("address", address).
		Debug("resource type not supported for cost estimation")
}
