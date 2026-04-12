package ec2

import (
	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcespec"
)

// DefaultEIPHourlyCost is $0.005/hr for public IPv4 (since Feb 2024).
const DefaultEIPHourlyCost = 0.005

type eipAttrs struct {
	Instance string
}

func parseEIPAttrs(attrs map[string]any) eipAttrs {
	return eipAttrs{
		Instance: handler.GetStringAttr(attrs, "instance"),
	}
}

// EIPSpec declares aws_eip cost estimation.
func EIPSpec(deps awskit.RuntimeDeps) resourcespec.ResourceSpec {
	vpcServiceID := deps.RuntimeOrDefault().MustService(awskit.ServiceKeyVPC)

	return resourcespec.ResourceSpec{
		Type:     handler.ResourceType(awskit.ResourceEIP),
		Category: handler.CostCategoryStandard,
		Lookup: &resourcespec.LookupSpec{
			BuildFunc: func(region string, attrs map[string]any) (*pricing.PriceLookup, error) {
				parsed := parseEIPAttrs(attrs)
				runtime := deps.RuntimeOrDefault()
				prefix := runtime.ResolveUsagePrefix(region)

				usagetype := prefix + "-PublicIPv4:InUseAddress"
				if parsed.Instance == "" {
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
		Describe: &resourcespec.DescribeSpec{
			BuildFunc: func(_ *pricing.Price, attrs map[string]any) map[string]string {
				details := map[string]string{}
				if parseEIPAttrs(attrs).Instance != "" {
					details["attached"] = "true"
				} else {
					details["attached"] = "false"
				}
				return details
			},
		},
		Standard: &resourcespec.StandardPricingSpec{
			CostFunc: func(price *pricing.Price, _ *pricing.PriceIndex, _ string, _ map[string]any) (hourly, monthly float64) {
				if price != nil && price.OnDemandUSD > 0 {
					return handler.HourlyCost(price.OnDemandUSD)
				}
				return handler.HourlyCost(DefaultEIPHourlyCost)
			},
		},
	}
}
