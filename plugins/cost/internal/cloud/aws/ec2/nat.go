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

type natAttrs struct {
	PublicIP         string
	ConnectivityType string
}

func parseNATAttrs(attrs map[string]any) natAttrs {
	ct := costutil.GetStringAttr(attrs, "connectivity_type")
	if ct == "" {
		ct = "public"
	}
	return natAttrs{
		PublicIP:         costutil.GetStringAttr(attrs, "public_ip"),
		ConnectivityType: ct,
	}
}

// NATSpec declares aws_nat_gateway cost estimation.
func NATSpec(deps awskit.RuntimeDeps) resourcespec.TypedSpec[natAttrs] {
	return resourcespec.TypedSpec[natAttrs]{
		Type:     resourcedef.ResourceType(awskit.ResourceNATGateway),
		Category: resourcedef.CostCategoryStandard,
		Parse:    parseNATAttrs,
		Lookup: &resourcespec.TypedLookupSpec[natAttrs]{
			BuildFunc: func(region string, _ natAttrs) (*pricing.PriceLookup, error) {
				return deps.RuntimeOrDefault().
					NewLookupBuilder(awskit.ServiceKeyEC2, "NAT Gateway").
					Attr("group", "NGW:NatGateway").
					Build(region), nil
			},
		},
		Describe: &resourcespec.TypedDescribeSpec[natAttrs]{
			BuildFunc: func(_ *pricing.Price, p natAttrs) map[string]string {
				return awskit.NewDescribeBuilder().
					String("connectivity", p.ConnectivityType).
					String("public_ip", p.PublicIP).
					Map()
			},
		},
		Standard: &resourcespec.TypedStandardPricingSpec[natAttrs]{
			CostFunc: func(price *pricing.Price, _ *pricing.PriceIndex, _ string, _ natAttrs) (hourly, monthly float64) {
				return awskit.NewCostBuilder().Hourly().Fallback(DefaultNATGatewayHourlyCost).Calc(price, nil, "")
			},
		},
	}
}
