package costengine

import (
	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
)

// PrefetchPlan collects the pricing indexes required to estimate one or more module plans.
type PrefetchPlan map[pricing.ServiceID]map[string]bool

// Add registers a service/region requirement.
func (p PrefetchPlan) Add(service pricing.ServiceID, region string) {
	if p[service] == nil {
		p[service] = make(map[string]bool)
	}
	p[service][region] = true
}

// Services returns the plan in cache.Validate/GetIndex compatible shape.
func (p PrefetchPlan) Services() map[pricing.ServiceID][]string {
	services := make(map[pricing.ServiceID][]string, len(p))
	for svc, regionSet := range p {
		for region := range regionSet {
			services[svc] = append(services[svc], region)
		}
	}
	return services
}

// PrefetchPlanner builds pricing warmup requirements from scanned module plans.
type PrefetchPlanner struct {
	router   ProviderRouter
	registry RegistryLookup
}

// NewPrefetchPlanner creates a prefetch planner backed by the handler registry.
func NewPrefetchPlanner(router ProviderRouter, registry RegistryLookup) *PrefetchPlanner {
	return &PrefetchPlanner{router: router, registry: registry}
}

// Build constructs the set of service/region indexes needed for the provided module plans.
func (p *PrefetchPlanner) Build(modulePlans []*ModulePlan) PrefetchPlan {
	required := make(PrefetchPlan)

	for _, modulePlan := range modulePlans {
		for _, resource := range modulePlan.Resources {
			providerID, ok := p.router.ResolveProvider(resource.ResourceType)
			if !ok {
				continue
			}

			h, ok := p.registry.ResolveHandler(providerID, resource.ResourceType)
			if !ok || h.Category() != handler.CostCategoryStandard {
				continue
			}

			lookupBuilder, ok := h.(handler.LookupBuilder)
			if !ok {
				continue
			}

			lookup, err := lookupBuilder.BuildLookup(modulePlan.Region, resource.ActiveAttrs())
			if err != nil || lookup == nil {
				continue
			}

			required.Add(lookup.ServiceID, modulePlan.Region)
		}
	}

	return required
}
