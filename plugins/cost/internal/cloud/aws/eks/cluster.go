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
type ClusterHandler struct{}

func (h *ClusterHandler) Category() handler.CostCategory { return handler.CostCategoryStandard }

func (h *ClusterHandler) ServiceCode() pricing.ServiceID {
	return awskit.MustService(awskit.ServiceKeyEKS)
}

func (h *ClusterHandler) Describe(_ *pricing.Price, attrs map[string]any) map[string]string {
	desc := make(map[string]string)
	if v := handler.GetStringAttr(attrs, "version"); v != "" {
		desc["version"] = v
	}
	return desc
}

func (h *ClusterHandler) BuildLookup(region string, _ map[string]any) (*pricing.PriceLookup, error) {
	prefix := awskit.ResolveUsagePrefix(region)

	lb := &awskit.LookupBuilder{Service: awskit.MustService(awskit.ServiceKeyEKS), ProductFamily: "Compute"}
	return lb.Build(region, map[string]string{
		"usagetype": prefix + "-AmazonEKS-Hours:perCluster",
	}), nil
}

func (h *ClusterHandler) CalculateCost(price *pricing.Price, _ *pricing.PriceIndex, _ string, _ map[string]any) (hourly, monthly float64) {
	if price != nil && price.OnDemandUSD > 0 {
		return handler.HourlyCost(price.OnDemandUSD)
	}
	// Fallback: $0.10/hr is consistent across all regions
	return handler.HourlyCost(DefaultClusterHourlyCost)
}
