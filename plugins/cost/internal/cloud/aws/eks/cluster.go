package eks

import (
	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/costutil"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcedef"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcespec"
)

// EKS pricing constants
const (
	DefaultClusterHourlyCost = 0.10
)

type clusterAttrs struct {
	Version string
}

func parseClusterAttrs(attrs map[string]any) clusterAttrs {
	return clusterAttrs{
		Version: costutil.GetStringAttr(attrs, "version"),
	}
}

// ClusterSpec declares aws_eks_cluster cost estimation.
func ClusterSpec(deps awskit.RuntimeDeps) resourcespec.ResourceSpec {
	return resourcespec.ResourceSpec{
		Type:     resourcedef.ResourceType(awskit.ResourceEKSCluster),
		Category: resourcedef.CostCategoryStandard,
		Lookup: &resourcespec.LookupSpec{
			BuildFunc: func(region string, _ map[string]any) (*pricing.PriceLookup, error) {
				runtime := deps.RuntimeOrDefault()
				prefix := runtime.ResolveUsagePrefix(region)
				return runtime.StandardLookupSpec(
					awskit.ServiceKeyEKS,
					"Compute",
					func(_ string, _ map[string]any) (map[string]string, error) {
						return map[string]string{
							"usagetype": prefix + "-AmazonEKS-Hours:perCluster",
						}, nil
					},
				).Build(region, nil)
			},
		},
		Describe: &resourcespec.DescribeSpec{
			Fields: []resourcespec.DescribeField{
				{Key: "version", Value: resourcespec.StringAttr("version"), OmitEmpty: true},
			},
		},
		Standard: &resourcespec.StandardPricingSpec{
			CostFunc: func(price *pricing.Price, _ *pricing.PriceIndex, _ string, _ map[string]any) (hourly, monthly float64) {
				if price != nil && price.OnDemandUSD > 0 {
					return costutil.HourlyCost(price.OnDemandUSD)
				}
				return costutil.HourlyCost(DefaultClusterHourlyCost)
			},
		},
	}
}
