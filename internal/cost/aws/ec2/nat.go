package ec2

import (
	aws "github.com/edelwud/terraci/internal/cost/aws"
	"github.com/edelwud/terraci/internal/cost/pricing"
)

// DefaultNATGatewayHourlyCost is the default NAT Gateway hourly rate.
const DefaultNATGatewayHourlyCost = 0.045

// NATHandler handles aws_nat_gateway cost estimation.
type NATHandler struct{}

func (h *NATHandler) Category() aws.CostCategory { return aws.CostCategoryStandard }

func (h *NATHandler) ServiceCode() pricing.ServiceCode {
	return pricing.ServiceEC2
}

func (h *NATHandler) BuildLookup(region string, _ map[string]any) (*pricing.PriceLookup, error) {
	// NAT Gateway pricing is in the EC2 service, not VPC
	lb := &aws.LookupBuilder{Service: pricing.ServiceEC2, ProductFamily: "NAT Gateway"}
	return lb.Build(region, map[string]string{
		"group": "NGW:NatGateway",
	}), nil
}

func (h *NATHandler) CalculateCost(price *pricing.Price, _ map[string]any) (hourly, monthly float64) {
	// NAT Gateway: hourly charge + data processing
	// For fixed cost estimation, only include hourly
	rate := price.OnDemandUSD
	if rate == 0 {
		rate = DefaultNATGatewayHourlyCost
	}
	return aws.HourlyCost(rate)
}
