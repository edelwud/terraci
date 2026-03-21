package ec2

import (
	aws "github.com/edelwud/terraci/internal/cost/aws"
	"github.com/edelwud/terraci/internal/cost/pricing"
)

// DefaultEIPHourlyCost is $0.005/hr for public IPv4 (since Feb 2024).
const DefaultEIPHourlyCost = 0.005

// EIPHandler handles aws_eip cost estimation.
type EIPHandler struct{}

func (h *EIPHandler) Category() aws.CostCategory { return aws.CostCategoryStandard }

func (h *EIPHandler) ServiceCode() pricing.ServiceCode {
	return pricing.ServiceVPC
}

func (h *EIPHandler) BuildLookup(region string, attrs map[string]any) (*pricing.PriceLookup, error) {
	// Since Feb 2024, AWS charges $0.005/hr for all public IPv4 addresses.
	// Pricing is under AmazonVPC service with productFamily "None".
	prefix := aws.ResolveUsagePrefix(region)

	usagetype := prefix + "-PublicIPv4:InUseAddress"
	if instance := aws.GetStringAttr(attrs, "instance"); instance == "" {
		usagetype = prefix + "-PublicIPv4:IdleAddress"
	}

	// AWS VPC pricing uses group "VPCPublicIPv4Address" and no product family.
	return &pricing.PriceLookup{
		ServiceCode: pricing.ServiceVPC,
		Region:      region,
		Attributes: map[string]string{
			"location":  aws.ResolveRegionName(region),
			"usagetype": usagetype,
			"group":     "VPCPublicIPv4Address",
		},
	}, nil
}

func (h *EIPHandler) CalculateCost(price *pricing.Price, _ map[string]any) (hourly, monthly float64) {
	if price != nil && price.OnDemandUSD > 0 {
		return aws.HourlyCost(price.OnDemandUSD)
	}
	// Fallback: $0.005/hr ($3.65/month) since Feb 2024
	// Attached to running instance still costs $0.005/hr
	return aws.HourlyCost(DefaultEIPHourlyCost)
}
