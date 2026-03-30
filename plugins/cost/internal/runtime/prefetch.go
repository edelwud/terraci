package runtime

import (
	"context"
	"fmt"

	"github.com/caarlos0/log"

	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
)

// ServicePlan exposes the minimal prefetch-plan shape consumed by the runtime prefetcher.
type ServicePlan interface {
	Services() map[pricing.ServiceID][]string
}

// PricingPrefetcher warms missing provider pricing indexes through the runtime registry.
type PricingPrefetcher struct {
	runtimes *ProviderRuntimeRegistry
}

// NewPricingPrefetcher creates a prefetch service backed by the provider runtime registry.
func NewPricingPrefetcher(runtimes *ProviderRuntimeRegistry) *PricingPrefetcher {
	return &PricingPrefetcher{runtimes: runtimes}
}

// PrefetchPricing downloads any missing pricing data required by the plan.
func (p *PricingPrefetcher) PrefetchPricing(ctx context.Context, prefetchPlan ServicePlan) error {
	services := prefetchPlan.Services()
	if len(services) == 0 {
		log.Warn("no supported resources found in plans - nothing to price")
		return nil
	}

	var totalMissing int
	for providerID, runtime := range p.runtimes.runtimes {
		providerServices := filterServicesForProvider(services, providerID)
		if len(providerServices) == 0 {
			continue
		}
		missing := runtime.Cache.Validate(providerServices)
		if len(missing) == 0 {
			continue
		}
		for i, m := range missing {
			totalMissing++
			log.WithField("provider", providerID).
				WithField("service", m.Service.String()).
				WithField("region", m.Region).
				WithField("progress", fmt.Sprintf("%d/%d", i+1, len(missing))).
				Info("downloading pricing data")
			if _, err := runtime.Cache.GetIndex(ctx, m.Service, m.Region); err != nil {
				return fmt.Errorf("fetch %s/%s pricing: %w", m.Service.String(), m.Region, err)
			}
		}
	}

	if totalMissing == 0 {
		log.Info("all pricing data is cached and valid")
	}

	return nil
}

func filterServicesForProvider(services map[pricing.ServiceID][]string, providerID string) map[pricing.ServiceID][]string {
	filtered := make(map[pricing.ServiceID][]string)
	for serviceID, regions := range services {
		if serviceID.Provider == providerID {
			filtered[serviceID] = regions
		}
	}
	return filtered
}
