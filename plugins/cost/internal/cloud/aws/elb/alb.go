package elb

import (
	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/costutil"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcedef"
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
		LoadBalancerType: costutil.GetStringAttr(attrs, "load_balancer_type"),
	}
	if parsed.LoadBalancerType == "" {
		parsed.LoadBalancerType = typeApplication
	}
	return parsed
}

// ALBSpec declares aws_lb/aws_alb cost estimation.
func ALBSpec(deps awskit.RuntimeDeps) resourcespec.TypedSpec[lbAttrs] {
	return resourcespec.TypedSpec[lbAttrs]{
		Type:     resourcedef.ResourceType(awskit.ResourceLoadBalancer),
		Category: resourcedef.CostCategoryStandard,
		Parse:    parseLBAttrs,
		Lookup: &resourcespec.TypedLookupSpec[lbAttrs]{
			BuildFunc: func(region string, p lbAttrs) (*pricing.PriceLookup, error) {
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
				switch p.LoadBalancerType {
				case typeNetwork:
					spec.ProductFamily = "Load Balancer-Network"
				case typeGateway:
					spec.ProductFamily = "Load Balancer-Gateway"
				default:
					spec.ProductFamily = productFamilyALB
				}
				return spec.Build(region, nil)
			},
		},
		Describe: &resourcespec.TypedDescribeSpec[lbAttrs]{
			BuildFunc: func(_ *pricing.Price, p lbAttrs) map[string]string {
				return map[string]string{"type": p.LoadBalancerType}
			},
		},
		Standard: &resourcespec.TypedStandardPricingSpec[lbAttrs]{
			CostFunc: func(price *pricing.Price, _ *pricing.PriceIndex, _ string, p lbAttrs) (hourly, monthly float64) {
				if price != nil && price.OnDemandUSD > 0 {
					return costutil.HourlyCost(price.OnDemandUSD)
				}
				switch p.LoadBalancerType {
				case typeNetwork:
					return costutil.HourlyCost(defaultNLBHourlyCost)
				case typeGateway:
					return costutil.HourlyCost(defaultGWLBHourlyCost)
				default:
					return costutil.HourlyCost(defaultALBHourlyCost)
				}
			},
		},
	}
}
