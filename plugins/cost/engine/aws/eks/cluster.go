package eks

import (
	aws "github.com/edelwud/terraci/plugins/cost/engine/aws"
	"github.com/edelwud/terraci/plugins/cost/engine/pricing"
)

// EKS pricing constants
const (
	DefaultClusterHourlyCost = 0.10
)

// ClusterHandler handles aws_eks_cluster cost estimation
type ClusterHandler struct{}

func (h *ClusterHandler) Category() aws.CostCategory { return aws.CostCategoryStandard }

func (h *ClusterHandler) ServiceCode() pricing.ServiceCode {
	return pricing.ServiceEKS
}

func (h *ClusterHandler) Describe(_ *pricing.Price, attrs map[string]any) map[string]string {
	desc := make(map[string]string)
	if v := aws.GetStringAttr(attrs, "version"); v != "" {
		desc["version"] = v
	}
	return desc
}

func (h *ClusterHandler) BuildLookup(region string, _ map[string]any) (*pricing.PriceLookup, error) {
	prefix := aws.ResolveUsagePrefix(region)

	lb := &aws.LookupBuilder{Service: pricing.ServiceEKS, ProductFamily: "Compute"}
	return lb.Build(region, map[string]string{
		"usagetype": prefix + "-AmazonEKS-Hours:perCluster",
	}), nil
}

func (h *ClusterHandler) CalculateCost(price *pricing.Price, _ *pricing.PriceIndex, _ string, _ map[string]any) (hourly, monthly float64) {
	if price != nil && price.OnDemandUSD > 0 {
		return aws.HourlyCost(price.OnDemandUSD)
	}
	// Fallback: $0.10/hr is consistent across all regions
	return aws.HourlyCost(DefaultClusterHourlyCost)
}
