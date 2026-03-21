package storage

import (
	aws "github.com/edelwud/terraci/internal/cost/aws"
	"github.com/edelwud/terraci/internal/cost/pricing"
)

const (
	Route53HostedZoneCost = 0.50
)

// Route53Handler handles aws_route53_zone cost estimation
type Route53Handler struct{}

func (h *Route53Handler) Category() aws.CostCategory { return aws.CostCategoryStandard }

func (h *Route53Handler) ServiceCode() pricing.ServiceCode {
	return pricing.ServiceRoute53
}

func (h *Route53Handler) BuildLookup(_ string, _ map[string]any) (*pricing.PriceLookup, error) {
	return &pricing.PriceLookup{
		ServiceCode:   pricing.ServiceRoute53,
		Region:        "global",
		ProductFamily: "DNS Zone",
		Attributes:    map[string]string{},
	}, nil
}

func (h *Route53Handler) CalculateCost(_ *pricing.Price, _ map[string]any) (hourly, monthly float64) {
	// Route53: $0.50 per hosted zone per month (first 25 zones)
	// Then $0.10 per zone after 25
	return aws.FixedMonthlyCost(Route53HostedZoneCost)
}
