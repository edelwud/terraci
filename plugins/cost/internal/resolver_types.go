package costengine

import (
	"context"

	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
)

// RegistryLookup is the minimal interface for finding resource handlers.
// Satisfied by *handler.Registry.
type RegistryLookup interface {
	GetHandler(resourceType handler.ResourceType) (handler.ResourceHandler, bool)
	Resolve(resourceType handler.ResourceType) (handler.RegisteredHandler, bool)
}

// PricingSource abstracts pricing index retrieval for cost resolution.
// Satisfied by estimator provider runtimes.
type PricingSource interface {
	GetIndex(ctx context.Context, service pricing.ServiceID, region string) (*pricing.PriceIndex, error)
	SourceName(providerID string) string
}

// ResolveRequest bundles all inputs for a single resource cost resolution.
type ResolveRequest struct {
	ResourceType handler.ResourceType
	Address      string
	Name         string
	ModuleAddr   string
	Region       string
	Attrs        map[string]any
}

// ResolveFunc is the signature of a cost resolution function.
type ResolveFunc func(ctx context.Context, req ResolveRequest) ResourceCost

// CostMiddleware wraps a cost resolution step.
// It receives the next resolver in the chain and the request, and can
// modify inputs, outputs, or short-circuit resolution entirely.
//
// Example: discount middleware that applies a multiplier to all costs:
//
//	func discountMiddleware(factor float64) CostMiddleware {
//	    return func(ctx context.Context, next ResolveFunc, req ResolveRequest) ResourceCost {
//	        result := next(ctx, req)
//	        result.HourlyCost *= factor
//	        result.MonthlyCost *= factor
//	        return result
//	    }
//	}
type CostMiddleware func(ctx context.Context, next ResolveFunc, req ResolveRequest) ResourceCost
