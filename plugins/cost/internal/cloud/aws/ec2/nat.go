package ec2

import (
	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcespec"
)

// DefaultNATGatewayHourlyCost is the default NAT Gateway hourly rate.
const DefaultNATGatewayHourlyCost = 0.045

// NATSpec declares aws_nat_gateway cost estimation.
func NATSpec(deps awskit.RuntimeDeps) resourcespec.ResourceSpec {
	return resourcespec.ResourceSpec{
		Type:     handler.ResourceType(awskit.ResourceNATGateway),
		Category: handler.CostCategoryStandard,
		Lookup: &resourcespec.LookupSpec{
			BuildFunc: func(region string, _ map[string]any) (*pricing.PriceLookup, error) {
				return deps.RuntimeOrDefault().StandardLookupSpec(
					awskit.ServiceKeyEC2,
					"NAT Gateway",
					func(_ string, _ map[string]any) (map[string]string, error) {
						return map[string]string{"group": "NGW:NatGateway"}, nil
					},
				).Build(region, nil)
			},
		},
		Describe: &resourcespec.DescribeSpec{
			BuildFunc: func(_ *pricing.Price, _ map[string]any) map[string]string {
				return map[string]string{}
			},
		},
		Standard: &resourcespec.StandardPricingSpec{
			CostFunc: func(price *pricing.Price, _ *pricing.PriceIndex, _ string, _ map[string]any) (hourly, monthly float64) {
				rate := DefaultNATGatewayHourlyCost
				if price != nil && price.OnDemandUSD > 0 {
					rate = price.OnDemandUSD
				}
				return handler.HourlyCost(rate)
			},
		},
	}
}
