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
	return resourcespec.TypedSpec[eipAttrs]{
		Type:     resourcedef.ResourceType(awskit.ResourceEIP),
		Category: resourcedef.CostCategoryStandard,
		Parse:    parseEIPAttrs,
		Lookup: &resourcespec.TypedLookupSpec[eipAttrs]{
			BuildFunc: func(region string, p eipAttrs) (*pricing.PriceLookup, error) {
				runtime := deps.RuntimeOrDefault()
				return runtime.
					NewLookupBuilder(awskit.ServiceKeyVPC, "").
					Attr("group", "VPCPublicIPv4Address").
					UsageType(region, awskit.MatchString(p.Instance, "PublicIPv4:InUseAddress", map[string]string{
						"": "PublicIPv4:IdleAddress",
					})).
					Build(region), nil
			},
		},
		Describe: &resourcespec.TypedDescribeSpec[eipAttrs]{
			BuildFunc: func(_ *pricing.Price, p eipAttrs) map[string]string {
				return awskit.NewDescribeBuilder().
					String("attached", map[bool]string{true: "true", false: "false"}[p.Instance != ""]).
					Map()
			},
		},
		Standard: &resourcespec.TypedStandardPricingSpec[eipAttrs]{
			CostFunc: func(price *pricing.Price, _ *pricing.PriceIndex, _ string, _ eipAttrs) (hourly, monthly float64) {
				return awskit.NewCostBuilder().Hourly().Fallback(DefaultEIPHourlyCost).Calc(price, nil, "")
			},
		},
	}
}
