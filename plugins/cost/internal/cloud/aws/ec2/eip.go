package ec2

import (
	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/costutil"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcedef"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcespec"
)

// DefaultEIPHourlyCost is $0.005/hr for public IPv4 (since Feb 2024).
const DefaultEIPHourlyCost = 0.005

type eipAttrs struct {
	Instance string
}

func parseEIPAttrs(attrs map[string]any) eipAttrs {
	return eipAttrs{
		Instance: costutil.GetStringAttr(attrs, "instance"),
	}
}

// EIPSpec declares aws_eip cost estimation.
func EIPSpec(deps awskit.RuntimeDeps) resourcespec.TypedSpec[eipAttrs] {
	vpcServiceID := deps.RuntimeOrDefault().MustService(awskit.ServiceKeyVPC)

	return resourcespec.TypedSpec[eipAttrs]{
		Type:     resourcedef.ResourceType(awskit.ResourceEIP),
		Category: resourcedef.CostCategoryStandard,
		Parse:    parseEIPAttrs,
		Lookup: &resourcespec.TypedLookupSpec[eipAttrs]{
			BuildFunc: func(region string, p eipAttrs) (*pricing.PriceLookup, error) {
				runtime := deps.RuntimeOrDefault()
				prefix := runtime.ResolveUsagePrefix(region)

				usagetype := prefix + "-PublicIPv4:InUseAddress"
				if p.Instance == "" {
					usagetype = prefix + "-PublicIPv4:IdleAddress"
				}

				return &pricing.PriceLookup{
					ServiceID: vpcServiceID,
					Region:    region,
					Attributes: map[string]string{
						"location":  runtime.ResolveRegionName(region),
						"usagetype": usagetype,
						"group":     "VPCPublicIPv4Address",
					},
				}, nil
			},
		},
		Describe: &resourcespec.TypedDescribeSpec[eipAttrs]{
			BuildFunc: func(_ *pricing.Price, p eipAttrs) map[string]string {
				details := map[string]string{}
				if p.Instance != "" {
					details["attached"] = "true"
				} else {
					details["attached"] = "false"
				}
				return details
			},
		},
		Standard: &resourcespec.TypedStandardPricingSpec[eipAttrs]{
			CostFunc: func(price *pricing.Price, _ *pricing.PriceIndex, _ string, _ eipAttrs) (hourly, monthly float64) {
				if price != nil && price.OnDemandUSD > 0 {
					return costutil.HourlyCost(price.OnDemandUSD)
				}
				return costutil.HourlyCost(DefaultEIPHourlyCost)
			},
		},
	}
}
