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
func ClusterSpec(deps awskit.RuntimeDeps) resourcespec.TypedSpec[clusterAttrs] {
	return resourcespec.TypedSpec[clusterAttrs]{
		Type:     resourcedef.ResourceType(awskit.ResourceEKSCluster),
		Category: resourcedef.CostCategoryStandard,
		Parse:    parseClusterAttrs,
		Lookup: &resourcespec.TypedLookupSpec[clusterAttrs]{
			BuildFunc: func(region string, _ clusterAttrs) (*pricing.PriceLookup, error) {
				return deps.RuntimeOrDefault().
					NewLookupBuilder(awskit.ServiceKeyEKS, "Compute").
					UsageType(region, "AmazonEKS-Hours:perCluster").
					Build(region), nil
			},
		},
		Describe: &resourcespec.TypedDescribeSpec[clusterAttrs]{
			BuildFunc: func(_ *pricing.Price, p clusterAttrs) map[string]string {
				return awskit.NewDescribeBuilder().
					String("version", p.Version).
					Map()
			},
		},
		Standard: &resourcespec.TypedStandardPricingSpec[clusterAttrs]{
			CostFunc: func(price *pricing.Price, _ *pricing.PriceIndex, _ string, _ clusterAttrs) (hourly, monthly float64) {
				return awskit.NewCostBuilder().Hourly().Fallback(DefaultClusterHourlyCost).Calc(price, nil, "")
			},
		},
	}
}
