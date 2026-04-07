package elb

import (
	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
)

// LB pricing constants
const (
	usageType        = "LoadBalancerUsage"
	typeApplication  = "application"
	typeNetwork      = "network"
	typeGateway      = "gateway"
	productFamilyALB = "Load Balancer-Application"

	// Default hourly costs
	defaultALBHourlyCost  = 0.0225
	defaultNLBHourlyCost  = 0.0225
	defaultGWLBHourlyCost = 0.0125
)

// ALBHandler handles aws_lb (ALB/NLB) cost estimation
type ALBHandler struct {
	awskit.RuntimeDeps
}

type lbAttrs struct {
	LoadBalancerType string
}

func parseLBAttrs(attrs map[string]any) lbAttrs {
	parsed := lbAttrs{
		LoadBalancerType: handler.GetStringAttr(attrs, "load_balancer_type"),
	}
	if parsed.LoadBalancerType == "" {
		parsed.LoadBalancerType = typeApplication
	}
	return parsed
}

func (h *ALBHandler) Category() handler.CostCategory { return handler.CostCategoryStandard }

func (h *ALBHandler) BuildLookup(region string, attrs map[string]any) (*pricing.PriceLookup, error) {
	parsed := parseLBAttrs(attrs)

	runtime := h.RuntimeOrDefault()
	spec := runtime.StandardLookupSpec(
		awskit.ServiceKeyEC2,
		"",
		func(region string, _ map[string]any) (map[string]string, error) {
			return map[string]string{
				"usagetype": runtime.ResolveUsagePrefix(region) + "-" + usageType,
			}, nil
		},
	)

	switch parsed.LoadBalancerType {
	case typeNetwork:
		spec.ProductFamily = "Load Balancer-Network"
	case typeGateway:
		spec.ProductFamily = "Load Balancer-Gateway"
	default:
		spec.ProductFamily = productFamilyALB
	}

	return spec.Build(region, attrs)
}

func (h *ALBHandler) Describe(_ *pricing.Price, attrs map[string]any) map[string]string {
	return map[string]string{"type": parseLBAttrs(attrs).LoadBalancerType}
}

func (h *ALBHandler) CalculateCost(price *pricing.Price, _ *pricing.PriceIndex, _ string, attrs map[string]any) (hourly, monthly float64) {
	rate := price.OnDemandUSD
	if rate == 0 {
		// Default pricing if lookup fails
		switch parseLBAttrs(attrs).LoadBalancerType {
		case typeNetwork:
			rate = defaultNLBHourlyCost
		case typeGateway:
			rate = defaultGWLBHourlyCost
		default:
			rate = defaultALBHourlyCost
		}
	}
	return handler.HourlyCost(rate)
}
