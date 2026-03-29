package costengine

import (
	"context"

	"github.com/caarlos0/log"

	"github.com/edelwud/terraci/internal/terraform/plan"
	"github.com/edelwud/terraci/plugins/cost/internal/aws"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
)

// resolveResourceCost calculates cost for a resource given its full attributes.
func (e *Estimator) resolveResourceCost(ctx context.Context, resourceType, address, name, moduleAddr, region string, attrs map[string]any) ResourceCost {
	result := ResourceCost{
		Address:    address,
		ModuleAddr: moduleAddr,
		Type:       resourceType,
		Name:       name,
		Region:     region,
	}

	handler, ok := e.registry.GetHandler(resourceType)
	if !ok {
		result.ErrorKind = CostErrorNoHandler
		result.ErrorDetail = "no handler"
		aws.LogUnsupported(resourceType, address)
		return result
	}

	if attrs == nil {
		attrs = make(map[string]any)
	}

	result.Details = handler.Describe(nil, attrs)

	switch handler.Category() {
	case aws.CostCategoryUsageBased:
		result.ErrorKind = CostErrorUsageBased
		result.ErrorDetail = "usage-based"
		result.PriceSource = "usage-based"
		return result

	case aws.CostCategoryFixed:
		hourly, monthly := handler.CalculateCost(nil, nil, region, attrs)
		result.HourlyCost = hourly
		result.MonthlyCost = monthly
		result.PriceSource = "fixed"
		return result

	case aws.CostCategoryStandard:
		return e.resolveStandardCost(ctx, handler, attrs, region, result)
	}

	return result
}

// resolveStandardCost handles the full pricing API lookup path.
func (e *Estimator) resolveStandardCost(ctx context.Context, handler aws.ResourceHandler, attrs map[string]any, region string, result ResourceCost) ResourceCost {
	lookup, err := handler.BuildLookup(region, attrs)
	if err != nil {
		result.ErrorKind = CostErrorLookupFailed
		result.ErrorDetail = err.Error()
		return result
	}

	if lookup == nil {
		return result
	}

	index, err := e.cache.GetIndex(ctx, lookup.ServiceCode, region)
	if err != nil {
		log.WithError(err).
			WithField("service", lookup.ServiceCode).
			WithField("region", region).
			Debug("failed to get pricing index")
		result.ErrorKind = CostErrorAPIFailure
		result.ErrorDetail = "pricing unavailable"
		return result
	}

	if index == nil {
		result.ErrorKind = CostErrorAPIFailure
		result.ErrorDetail = "empty pricing index"
		return result
	}

	price, err := index.LookupPrice(*lookup)
	if err != nil {
		log.WithError(err).
			WithField("address", result.Address).
			Debug("price lookup failed")
		result.ErrorKind = CostErrorNoPrice
		result.ErrorDetail = "no matching price"
		return result
	}

	hourly, monthly := handler.CalculateCost(price, index, region, attrs)
	result.HourlyCost = hourly
	result.MonthlyCost = monthly
	result.PriceSource = "aws-bulk-api"
	result.Details = handler.Describe(price, attrs)

	return result
}

// resourceChange is a type alias to avoid leaking internal/terraform/plan into the resolver interface.
type resourceChange = plan.ResourceChange

// collectRequiredServices determines which AWS services need pricing data.
func (e *Estimator) collectRequiredServices(resources []resourceChange, region string) map[pricing.ServiceCode][]string {
	services := make(map[pricing.ServiceCode]map[string]bool)

	for _, rc := range resources {
		handler, ok := e.registry.GetHandler(rc.Type)
		if !ok || handler.Category() != aws.CostCategoryStandard {
			continue
		}

		svc := handler.ServiceCode()
		if services[svc] == nil {
			services[svc] = make(map[string]bool)
		}
		services[svc][region] = true
	}

	result := make(map[pricing.ServiceCode][]string)
	for svc, regionMap := range services {
		for r := range regionMap {
			result[svc] = append(result[svc], r)
		}
	}

	return result
}
