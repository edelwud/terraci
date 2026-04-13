package elasticache

import (
	"errors"

	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"

	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcedef"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcespec"
)

// ReplicationGroupSpec declares aws_elasticache_replication_group cost estimation.
func ReplicationGroupSpec(deps awskit.RuntimeDeps) resourcespec.TypedSpec[replicationGroupAttrs] {
	return resourcespec.TypedSpec[replicationGroupAttrs]{
		Type:     resourcedef.ResourceType(awskit.ResourceElastiCacheReplicationGroup),
		Category: resourcedef.CostCategoryStandard,
		Parse:    parseReplicationGroupAttrs,
		Lookup: &resourcespec.TypedLookupSpec[replicationGroupAttrs]{
			BuildFunc: func(region string, p replicationGroupAttrs) (*pricing.PriceLookup, error) {
				if p.NodeType == "" {
					return nil, errors.New("node_type not found")
				}
				runtime := deps.RuntimeOrDefault()
				return runtime.
					NewLookupBuilder(awskit.ServiceKeyElastiCache, "Cache Instance").
					Attr("instanceType", p.NodeType).
					Attr("cacheEngine", "Redis").
					UsageType(region, "NodeUsage:"+p.NodeType).
					Build(region), nil
			},
		},
		Describe: &resourcespec.TypedDescribeSpec[replicationGroupAttrs]{
			BuildFunc: func(price *pricing.Price, p replicationGroupAttrs) map[string]string {
				memoryGiB := 0.0
				ssdGiB := 0.0
				if price != nil {
					memoryGiB = awskit.ParseGiB(price.Attributes["memory"])
					ssdGiB = awskit.ParseGiB(price.Attributes["storage"])
				}
				return awskit.NewDescribeBuilder().
					String("node_type", p.NodeType).
					IntIf(p.NumNodeGroupsSet, "node_groups", p.NumNodeGroups).
					Int("replicas_per_group", p.ReplicasPerNodeGroup).
					Int("snapshot_retention_days", p.SnapshotRetentionDays).
					Int("total_nodes", p.totalNodes()).
					FloatIf(memoryGiB > 0, "memory_gib", memoryGiB, "%.2f").
					FloatIf(ssdGiB > 0, "ssd_gib", ssdGiB, "%.0f").
					Map()
			},
		},
		Standard: &resourcespec.TypedStandardPricingSpec[replicationGroupAttrs]{
			CostFunc: func(price *pricing.Price, index *pricing.PriceIndex, region string, p replicationGroupAttrs) (hourly, monthly float64) {
				rt := deps.RuntimeOrDefault()
				nodes := p.totalNodes()
				ssdGB := priceGiB(price, "storage")
				memGB := priceGiB(price, "memory")
				backupGB := memGB * float64(nodes) * float64(max(p.SnapshotRetentionDays-1, 0))

				return awskit.NewCostBuilder().
					Hourly().
					Scale(float64(nodes)).
					Charge(awskit.NewCharge(ssdGB*float64(nodes)).
						Rate(awskit.IndexRate(rt, "Cache Storage", rt.ResolveUsagePrefix(region)+"-DataTiering:StorageUsage")).
						Fallback(FallbackDataTieringCostPerGBMonth)).
					Charge(awskit.NewCharge(backupGB).
						Rate(awskit.IndexRate(rt, "Storage Snapshot", rt.ResolveUsagePrefix(region)+"-BackupUsage")).
						Fallback(FallbackBackupStorageCostPerGBMonth)).
					Calc(price, index, region)
			},
		},
	}
}
