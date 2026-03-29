package ec2

import (
	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
)

// DefaultNATGatewayHourlyCost is the default NAT Gateway hourly rate.
const DefaultNATGatewayHourlyCost = 0.045

// NATHandler handles aws_nat_gateway cost estimation.
type NATHandler struct {
	awskit.RuntimeDeps
}

func (h *NATHandler) Category() handler.CostCategory { return handler.CostCategoryStandard }

func (h *NATHandler) BuildLookup(region string, _ map[string]any) (*pricing.PriceLookup, error) {
	// NAT Gateway pricing is in the EC2 service, not VPC
	return h.RuntimeOrDefault().StandardLookupSpec(
		awskit.ServiceKeyEC2,
		"NAT Gateway",
		func(_ string, _ map[string]any) (map[string]string, error) {
			return map[string]string{
				"group": "NGW:NatGateway",
			}, nil
		},
	).Build(region, nil)
}

func (h *NATHandler) Describe(_ *pricing.Price, _ map[string]any) map[string]string {
	return map[string]string{}
}

func (h *NATHandler) CalculateCost(price *pricing.Price, _ *pricing.PriceIndex, _ string, _ map[string]any) (hourly, monthly float64) {
	// NAT Gateway: hourly charge + data processing
	// For fixed cost estimation, only include hourly
	rate := price.OnDemandUSD
	if rate == 0 {
		rate = DefaultNATGatewayHourlyCost
	}
	return handler.HourlyCost(rate)
}
