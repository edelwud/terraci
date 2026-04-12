package elb

import (
	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcespec"
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

// ALBSpec declares aws_lb/aws_alb cost estimation.
func ALBSpec(deps awskit.RuntimeDeps) resourcespec.ResourceSpec {
	return resourcespec.ResourceSpec{
		Type:     handler.ResourceType(awskit.ResourceLoadBalancer),
		Category: handler.CostCategoryStandard,
		Lookup: &resourcespec.LookupSpec{
			BuildFunc: func(region string, attrs map[string]any) (*pricing.PriceLookup, error) {
				parsed := parseLBAttrs(attrs)
				runtime := deps.RuntimeOrDefault()
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
			},
		},
		Describe: &resourcespec.DescribeSpec{
			Fields: []resourcespec.DescribeField{
				{Key: "type", Value: func(_ *pricing.Price, attrs map[string]any) (string, bool) {
					return parseLBAttrs(attrs).LoadBalancerType, true
				}},
			},
		},
		Standard: &resourcespec.StandardPricingSpec{
			CostFunc: func(price *pricing.Price, _ *pricing.PriceIndex, _ string, attrs map[string]any) (hourly, monthly float64) {
				if price != nil && price.OnDemandUSD > 0 {
					return handler.HourlyCost(price.OnDemandUSD)
				}
				switch parseLBAttrs(attrs).LoadBalancerType {
				case typeNetwork:
					return handler.HourlyCost(defaultNLBHourlyCost)
				case typeGateway:
					return handler.HourlyCost(defaultGWLBHourlyCost)
				default:
					return handler.HourlyCost(defaultALBHourlyCost)
				}
			},
		},
	}
}
