package ec2

import (
	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/costutil"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcedef"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcespec"
)

// DefaultNATGatewayHourlyCost is the default NAT Gateway hourly rate.
const DefaultNATGatewayHourlyCost = 0.045

// NATSpec declares aws_nat_gateway cost estimation.
func NATSpec(deps awskit.RuntimeDeps) resourcespec.TypedSpec[resourcespec.NoAttrs] {
	return resourcespec.TypedSpec[resourcespec.NoAttrs]{
		Type:     resourcedef.ResourceType(awskit.ResourceNATGateway),
		Category: resourcedef.CostCategoryStandard,
		Parse:    resourcespec.ParseNoAttrs,
		Lookup: &resourcespec.TypedLookupSpec[resourcespec.NoAttrs]{
			BuildFunc: func(region string, _ resourcespec.NoAttrs) (*pricing.PriceLookup, error) {
				return deps.RuntimeOrDefault().StandardLookupSpec(
					awskit.ServiceKeyEC2,
					"NAT Gateway",
					func(_ string, _ map[string]any) (map[string]string, error) {
						return map[string]string{"group": "NGW:NatGateway"}, nil
					},
				).Build(region, nil)
			},
		},
		Describe: &resourcespec.TypedDescribeSpec[resourcespec.NoAttrs]{
			BuildFunc: func(_ *pricing.Price, _ resourcespec.NoAttrs) map[string]string {
				return map[string]string{}
			},
		},
		Standard: &resourcespec.TypedStandardPricingSpec[resourcespec.NoAttrs]{
			CostFunc: func(price *pricing.Price, _ *pricing.PriceIndex, _ string, _ resourcespec.NoAttrs) (hourly, monthly float64) {
				rate := DefaultNATGatewayHourlyCost
				if price != nil && price.OnDemandUSD > 0 {
					rate = price.OnDemandUSD
				}
				return costutil.HourlyCost(rate)
			},
		},
	}
}
