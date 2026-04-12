package storage

import (
	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/costutil"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcedef"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcespec"
)

const (
	// Route53HostedZoneCost is the monthly cost per hosted zone (first 25 zones).
	Route53HostedZoneCost = 0.50
)

// Route53Spec declares aws_route53_zone cost estimation.
func Route53Spec() resourcespec.ResourceSpec {
	return resourcespec.ResourceSpec{
		Type:     resourcedef.ResourceType(awskit.ResourceRoute53Zone),
		Category: resourcedef.CostCategoryFixed,
		Lookup: &resourcespec.LookupSpec{
			BuildFunc: func(_ string, _ map[string]any) (*pricing.PriceLookup, error) { return nil, nil },
		},
		Describe: &resourcespec.DescribeSpec{
			BuildFunc: func(_ *pricing.Price, _ map[string]any) map[string]string { return nil },
		},
		Fixed: &resourcespec.FixedPricingSpec{
			CostFunc: func(_ string, _ map[string]any) (hourly, monthly float64) {
				return costutil.FixedMonthlyCost(Route53HostedZoneCost)
			},
		},
	}
}
