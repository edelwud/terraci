package storage //nolint:dupl // Route53 and SecretsManager are structurally similar fixed-cost handlers

import (
	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
)

const (
	// Route53HostedZoneCost is the monthly cost per hosted zone (first 25 zones).
	Route53HostedZoneCost = 0.50
)

// Route53Handler handles aws_route53_zone cost estimation
type Route53Handler struct{}

func (h *Route53Handler) Category() handler.CostCategory { return handler.CostCategoryFixed }

func (h *Route53Handler) ServiceCode() pricing.ServiceID {
	return awskit.MustService(awskit.ServiceKeyRoute53)
}

func (h *Route53Handler) BuildLookup(_ string, _ map[string]any) (*pricing.PriceLookup, error) {
	return nil, nil
}

func (h *Route53Handler) Describe(_ *pricing.Price, _ map[string]any) map[string]string { return nil }

func (h *Route53Handler) CalculateCost(_ *pricing.Price, _ *pricing.PriceIndex, _ string, _ map[string]any) (hourly, monthly float64) {
	// Route53: $0.50 per hosted zone per month (first 25 zones)
	return handler.FixedMonthlyCost(Route53HostedZoneCost)
}
