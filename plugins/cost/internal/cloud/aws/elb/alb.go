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
				builder := runtime.
					NewLookupBuilder(awskit.ServiceKeyEC2, productFamilyALB).
					UsageType(region, usageType).
					ProductFamilyMatch(p.LoadBalancerType, productFamilyALB, map[string]string{
						typeNetwork: "Load Balancer-Network",
						typeGateway: "Load Balancer-Gateway",
					})
				return builder.Build(region), nil
			},
		},
		Describe: &resourcespec.TypedDescribeSpec[lbAttrs]{
			BuildFunc: func(_ *pricing.Price, p lbAttrs) map[string]string {
				return map[string]string{"type": p.LoadBalancerType}
			},
		},
		Standard: &resourcespec.TypedStandardPricingSpec[lbAttrs]{
			CostFunc: func(price *pricing.Price, _ *pricing.PriceIndex, _ string, p lbAttrs) (hourly, monthly float64) {
				fallback := defaultALBHourlyCost
				switch p.LoadBalancerType {
				case typeNetwork:
					fallback = defaultNLBHourlyCost
				case typeGateway:
					fallback = defaultGWLBHourlyCost
				}
				return awskit.NewCostBuilder().Hourly().Fallback(fallback).Calc(price, nil, "")
			},
		},
	}
}
