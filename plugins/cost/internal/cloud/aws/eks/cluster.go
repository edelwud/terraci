package eks

import (
	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
)

// EKS pricing constants
const (
	DefaultClusterHourlyCost = 0.10
)

// ClusterHandler handles aws_eks_cluster cost estimation
type ClusterHandler struct {
	awskit.RuntimeDeps
}

type clusterAttrs struct {
	Version string
}

func parseClusterAttrs(attrs map[string]any) clusterAttrs {
	return clusterAttrs{
		Version: handler.GetStringAttr(attrs, "version"),
	}
}

func (h *ClusterHandler) Category() handler.CostCategory { return handler.CostCategoryStandard }

func (h *ClusterHandler) Describe(_ *pricing.Price, attrs map[string]any) map[string]string {
	parsed := parseClusterAttrs(attrs)
	return awskit.NewDescribeBuilder().
		String("version", parsed.Version).
		Map()
}

func (h *ClusterHandler) BuildLookup(region string, _ map[string]any) (*pricing.PriceLookup, error) {
	runtime := h.RuntimeOrDefault()
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
}

func (h *ClusterHandler) CalculateCost(price *pricing.Price, _ *pricing.PriceIndex, _ string, _ map[string]any) (hourly, monthly float64) {
	if price != nil && price.OnDemandUSD > 0 {
		return handler.HourlyCost(price.OnDemandUSD)
	}
	// Fallback: $0.10/hr is consistent across all regions
	return handler.HourlyCost(DefaultClusterHourlyCost)
}
