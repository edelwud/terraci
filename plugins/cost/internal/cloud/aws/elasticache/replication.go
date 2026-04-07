package elasticache

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
)

// ReplicationGroupHandler handles aws_elasticache_replication_group cost estimation
type ReplicationGroupHandler struct {
	awskit.RuntimeDeps
}

func (h *ReplicationGroupHandler) Category() handler.CostCategory {
	return handler.CostCategoryStandard
}

func (h *ReplicationGroupHandler) BuildLookup(region string, attrs map[string]any) (*pricing.PriceLookup, error) {
	parsed := parseReplicationGroupAttrs(attrs)
	if parsed.NodeType == "" {
		return nil, errors.New("node_type not found")
	}

	runtime := h.RuntimeOrDefault()
	spec := runtime.StandardLookupSpec(
		awskit.ServiceKeyElastiCache,
		"Cache Instance",
		func(region string, _ map[string]any) (map[string]string, error) {
			prefix := runtime.ResolveUsagePrefix(region)
			return map[string]string{
				"instanceType": parsed.NodeType,
				"cacheEngine":  "Redis",
				"usagetype":    prefix + "-NodeUsage:" + parsed.NodeType,
			}, nil
		},
	)

	return spec.Build(region, attrs)
}

func (h *ReplicationGroupHandler) Describe(price *pricing.Price, attrs map[string]any) map[string]string {
	parsed := parseReplicationGroupAttrs(attrs)
	desc := make(map[string]string)
	if parsed.NodeType != "" {
		desc["node_type"] = parsed.NodeType
	}
	if parsed.NumNodeGroupsSet {
		desc["node_groups"] = strconv.Itoa(parsed.NumNodeGroups)
	}
	if parsed.ReplicasPerNodeGroup != 0 {
		desc["replicas_per_group"] = strconv.Itoa(parsed.ReplicasPerNodeGroup)
	}
	if parsed.SnapshotRetentionDays > 0 {
		desc["snapshot_retention_days"] = strconv.Itoa(parsed.SnapshotRetentionDays)
	}
	totalNodes := parsed.totalNodes()
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
	parsed := parseReplicationGroupAttrs(attrs)
	totalNodes := parsed.totalNodes()

	_, monthly = handler.ScaledHourlyCost(price.OnDemandUSD, totalNodes)

	// Data tiering cost for nodes with local SSD (r6gd/r7gd)
	monthly += dataTieringCost(h.RuntimeOrDefault(), price, index, region, totalNodes)

	// Backup storage cost
	if parsed.SnapshotRetentionDays > 0 {
		monthly += backupStorageCost(h.RuntimeOrDefault(), price, index, region, totalNodes, parsed.SnapshotRetentionDays)
	}

	hourly = monthly / handler.HoursPerMonth
	return hourly, monthly
}
