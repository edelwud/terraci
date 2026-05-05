package elb

import (
	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcedef"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcespec"
)

const (
	DefaultClassicLBHourlyCost = 0.025

	// describeKeyType is the Describe payload key for load-balancer type.
	describeKeyType = "type"
)

// ClassicSpec declares aws_elb cost estimation.
func ClassicSpec(deps awskit.RuntimeDeps) resourcespec.TypedSpec[resourcespec.NoAttrs] {
	return resourcespec.TypedSpec[resourcespec.NoAttrs]{
		Type:     resourcedef.ResourceType(awskit.ResourceClassicLoadBalancer),
		Category: resourcedef.CostCategoryStandard,
		Parse:    resourcespec.ParseNoAttrs,
		Lookup: &resourcespec.TypedLookupSpec[resourcespec.NoAttrs]{
			BuildFunc: func(region string, _ resourcespec.NoAttrs) (*pricing.PriceLookup, error) {
				runtime := deps.RuntimeOrDefault()
				return runtime.
					NewLookupBuilder(awskit.ServiceKeyELB, "Load Balancer").
					UsageType(region, usageType).
					Build(region), nil
			},
		},
		Describe: &resourcespec.TypedDescribeSpec[resourcespec.NoAttrs]{
			BuildFunc: func(_ *pricing.Price, _ resourcespec.NoAttrs) map[string]string {
				return map[string]string{describeKeyType: "classic"}
			},
		},
		Standard: &resourcespec.TypedStandardPricingSpec[resourcespec.NoAttrs]{
			CostFunc: func(price *pricing.Price, _ *pricing.PriceIndex, _ string, _ resourcespec.NoAttrs) (hourly, monthly float64) {
				return awskit.NewCostBuilder().Hourly().Fallback(DefaultClassicLBHourlyCost).Calc(price, nil, "")
			},
		},
	}
}
