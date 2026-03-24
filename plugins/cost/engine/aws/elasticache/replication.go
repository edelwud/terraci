package elasticache

import (
	"fmt"
	"strconv"

	aws "github.com/edelwud/terraci/plugins/cost/engine/aws"
	"github.com/edelwud/terraci/plugins/cost/engine/pricing"
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

func (h *ReplicationGroupHandler) Describe(price *pricing.Price, attrs map[string]any) map[string]string {
	desc := make(map[string]string)
	if v := aws.GetStringAttr(attrs, "node_type"); v != "" {
		desc["node_type"] = v
	}
	if v := aws.GetIntAttr(attrs, "num_node_groups"); v != 0 {
		desc["node_groups"] = strconv.Itoa(v)
	}
	if v := aws.GetIntAttr(attrs, "replicas_per_node_group"); v != 0 {
		desc["replicas_per_group"] = strconv.Itoa(v)
	}
	if v := aws.GetIntAttr(attrs, "snapshot_retention_limit"); v > 0 {
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

	_, monthly = aws.ScaledHourlyCost(price.OnDemandUSD, totalNodes)

	// Data tiering cost for nodes with local SSD (r6gd/r7gd)
	monthly += dataTieringCost(price, index, region, totalNodes)

	// Backup storage cost
	snapshotRetention := aws.GetIntAttr(attrs, "snapshot_retention_limit")
	if snapshotRetention > 0 {
		monthly += backupStorageCost(price, index, region, totalNodes, snapshotRetention)
	}

	hourly = monthly / aws.HoursPerMonth
	return hourly, monthly
}

// replicationGroupNodeCount calculates total node count from replication group attributes.
func replicationGroupNodeCount(attrs map[string]any) int {
	numNodeGroups := aws.GetIntAttr(attrs, "num_node_groups")
	if numNodeGroups == 0 {
		numNodeGroups = 1
	}

	replicasPerGroup := aws.GetIntAttr(attrs, "replicas_per_node_group")
	totalNodes := numNodeGroups * (1 + replicasPerGroup)

	// Legacy attribute support
	if totalNodes == 1 {
		if n := aws.GetIntAttr(attrs, "number_cache_clusters"); n > 0 {
			totalNodes = n
		}
	}

	return totalNodes
}
