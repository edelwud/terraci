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
func Route53Spec() resourcespec.TypedSpec[resourcespec.NoAttrs] {
	return resourcespec.TypedSpec[resourcespec.NoAttrs]{
		Type:     resourcedef.ResourceType(awskit.ResourceRoute53Zone),
		Category: resourcedef.CostCategoryFixed,
		Parse:    resourcespec.ParseNoAttrs,
		Lookup: &resourcespec.TypedLookupSpec[resourcespec.NoAttrs]{
			BuildFunc: func(_ string, _ resourcespec.NoAttrs) (*pricing.PriceLookup, error) { return nil, nil },
		},
		Describe: &resourcespec.TypedDescribeSpec[resourcespec.NoAttrs]{
			BuildFunc: func(_ *pricing.Price, _ resourcespec.NoAttrs) map[string]string { return nil },
		},
		Fixed: &resourcespec.TypedFixedPricingSpec[resourcespec.NoAttrs]{
			CostFunc: func(_ string, _ resourcespec.NoAttrs) (hourly, monthly float64) {
				return costutil.FixedMonthlyCost(Route53HostedZoneCost)
			},
		},
	}
}
