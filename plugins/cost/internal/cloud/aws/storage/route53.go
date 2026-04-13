package storage

import (
	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcedef"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcespec"
)

const (
	// Route53HostedZoneCost is the monthly cost per hosted zone (first 25 zones).
	Route53HostedZoneCost = 0.50
)

// Route53Spec declares aws_route53_zone cost estimation.
func Route53Spec() resourcespec.TypedSpec[resourcespec.NoAttrs] {
	return resourcespec.FixedMonthlyNoAttrsSpec(resourcedef.ResourceType(awskit.ResourceRoute53Zone), Route53HostedZoneCost)
}
