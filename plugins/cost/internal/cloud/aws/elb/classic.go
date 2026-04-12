package elb

import (
	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcespec"
)

const (
	DefaultClassicLBHourlyCost = 0.025
)

// ClassicSpec declares aws_elb cost estimation.
func ClassicSpec(deps awskit.RuntimeDeps) resourcespec.ResourceSpec {
	return resourcespec.ResourceSpec{
		Type:     handler.ResourceType(awskit.ResourceClassicLoadBalancer),
		Category: handler.CostCategoryStandard,
		Lookup: &resourcespec.LookupSpec{
			BuildFunc: func(region string, _ map[string]any) (*pricing.PriceLookup, error) {
				runtime := deps.RuntimeOrDefault()
				return runtime.StandardLookupSpec(
					awskit.ServiceKeyELB,
					"Load Balancer",
					func(region string, _ map[string]any) (map[string]string, error) {
						return map[string]string{
							"usagetype": runtime.ResolveUsagePrefix(region) + "-" + usageType,
						}, nil
					},
				).Build(region, nil)
			},
		},
		Describe: &resourcespec.DescribeSpec{
			Fields: []resourcespec.DescribeField{
				{Key: "type", Value: resourcespec.Const("classic")},
			},
		},
		Standard: &resourcespec.StandardPricingSpec{
			CostFunc: func(price *pricing.Price, _ *pricing.PriceIndex, _ string, _ map[string]any) (hourly, monthly float64) {
				if price != nil && price.OnDemandUSD > 0 {
					return handler.HourlyCost(price.OnDemandUSD)
				}
				return handler.HourlyCost(DefaultClassicLBHourlyCost)
			},
		},
	}
}
