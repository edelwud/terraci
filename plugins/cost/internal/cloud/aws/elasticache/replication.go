package elasticache

import (
	"fmt"
	"strconv"

	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
)

// ReplicationGroupHandler handles aws_elasticache_replication_group cost estimation
type ReplicationGroupHandler struct{}

func (h *ReplicationGroupHandler) Category() handler.CostCategory {
	return handler.CostCategoryStandard
}

func (h *ReplicationGroupHandler) ServiceCode() pricing.ServiceCode {
	return pricing.ServiceElastiCache
}

func (h *ReplicationGroupHandler) BuildLookup(region string, attrs map[string]any) (*pricing.PriceLookup, error) {
	nodeType := handler.GetStringAttr(attrs, "node_type")
	if nodeType == "" {
		return nil, fmt.Errorf("node_type not found")
	}

	prefix := awskit.ResolveUsagePrefix(region)
	usagetype := prefix + "-NodeUsage:" + nodeType

	lb := &awskit.LookupBuilder{Service: pricing.ServiceElastiCache, ProductFamily: "Cache Instance"}
	return lb.Build(region, map[string]string{
		"instanceType": nodeType,
		"cacheEngine":  "Redis",
		"usagetype":    usagetype,
	}), nil
}

func (h *ReplicationGroupHandler) Describe(price *pricing.Price, attrs map[string]any) map[string]string {
	desc := make(map[string]string)
	if v := handler.GetStringAttr(attrs, "node_type"); v != "" {
		desc["node_type"] = v
	}
	if v := handler.GetIntAttr(attrs, "num_node_groups"); v != 0 {
		desc["node_groups"] = strconv.Itoa(v)
	}
	if v := handler.GetIntAttr(attrs, "replicas_per_node_group"); v != 0 {
		desc["replicas_per_group"] = strconv.Itoa(v)
	}
	if v := handler.GetIntAttr(attrs, "snapshot_retention_limit"); v > 0 {
		desc["snapshot_retention_days"] = strconv.Itoa(v)
	}
	totalNodes := replicationGroupNodeCount(attrs)
	if totalNodes > 0 {
		desc["total_nodes"] = strconv.Itoa(totalNodes)
	}
	if mem := nodeMemoryFromPrice(price); mem > 0 {
		desc["memory_gib"] = fmt.Sprintf("%.2f", mem)
	}
	if ssd := nodeSSDFromPrice(price); ssd > 0 {
		desc["ssd_gib"] = fmt.Sprintf("%.0f", ssd)
	}
	return desc
}

func (h *ReplicationGroupHandler) CalculateCost(price *pricing.Price, index *pricing.PriceIndex, region string, attrs map[string]any) (hourly, monthly float64) {
	totalNodes := replicationGroupNodeCount(attrs)

	_, monthly = handler.ScaledHourlyCost(price.OnDemandUSD, totalNodes)

	// Data tiering cost for nodes with local SSD (r6gd/r7gd)
	monthly += dataTieringCost(price, index, region, totalNodes)

	// Backup storage cost
	snapshotRetention := handler.GetIntAttr(attrs, "snapshot_retention_limit")
	if snapshotRetention > 0 {
		monthly += backupStorageCost(price, index, region, totalNodes, snapshotRetention)
	}

	hourly = monthly / handler.HoursPerMonth
	return hourly, monthly
}

// replicationGroupNodeCount calculates total node count from replication group attributes.
func replicationGroupNodeCount(attrs map[string]any) int {
	numNodeGroups := handler.GetIntAttr(attrs, "num_node_groups")
	if numNodeGroups == 0 {
		numNodeGroups = 1
	}

	replicasPerGroup := handler.GetIntAttr(attrs, "replicas_per_node_group")
	totalNodes := numNodeGroups * (1 + replicasPerGroup)

	// Legacy attribute support
	if totalNodes == 1 {
		if n := handler.GetIntAttr(attrs, "number_cache_clusters"); n > 0 {
			totalNodes = n
		}
	}

	return totalNodes
}
