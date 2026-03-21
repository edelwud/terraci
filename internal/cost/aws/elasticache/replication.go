package elasticache

import (
	"fmt"

	aws "github.com/edelwud/terraci/internal/cost/aws"
	"github.com/edelwud/terraci/internal/cost/pricing"
)

// ReplicationGroupHandler handles aws_elasticache_replication_group cost estimation
type ReplicationGroupHandler struct{}

func (h *ReplicationGroupHandler) Category() aws.CostCategory { return aws.CostCategoryStandard }

func (h *ReplicationGroupHandler) ServiceCode() pricing.ServiceCode {
	return pricing.ServiceElastiCache
}

func (h *ReplicationGroupHandler) BuildLookup(region string, attrs map[string]any) (*pricing.PriceLookup, error) {
	nodeType := aws.GetStringAttr(attrs, "node_type")
	if nodeType == "" {
		return nil, fmt.Errorf("node_type not found")
	}

	prefix := aws.ResolveUsagePrefix(region)
	usagetype := prefix + "-NodeUsage:" + nodeType

	lb := &aws.LookupBuilder{Service: pricing.ServiceElastiCache, ProductFamily: "Cache Instance"}
	return lb.Build(region, map[string]string{
		"instanceType": nodeType,
		"cacheEngine":  "Redis",
		"usagetype":    usagetype,
	}), nil
}

func (h *ReplicationGroupHandler) CalculateCost(price *pricing.Price, attrs map[string]any) (hourly, monthly float64) {
	// Calculate total nodes: num_node_groups * replicas_per_node_group
	numNodeGroups := aws.GetIntAttr(attrs, "num_node_groups")
	if numNodeGroups == 0 {
		numNodeGroups = 1
	}

	replicasPerGroup := aws.GetIntAttr(attrs, "replicas_per_node_group")
	// Total nodes = primary (1 per group) + replicas
	totalNodes := numNodeGroups * (1 + replicasPerGroup)

	// Legacy attribute support
	if totalNodes == 1 {
		numberCacheClusters := aws.GetIntAttr(attrs, "number_cache_clusters")
		if numberCacheClusters > 0 {
			totalNodes = numberCacheClusters
		}
	}

	return aws.ScaledHourlyCost(price.OnDemandUSD, totalNodes)
}
