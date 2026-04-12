package elasticache

import (
	"errors"

	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcespec"
)

// ReplicationGroupSpec declares aws_elasticache_replication_group cost estimation.
func ReplicationGroupSpec(deps awskit.RuntimeDeps) resourcespec.ResourceSpec {
	return resourcespec.ResourceSpec{
		Type:     handler.ResourceType(awskit.ResourceElastiCacheReplicationGroup),
		Category: handler.CostCategoryStandard,
		Lookup: &resourcespec.LookupSpec{
			BuildFunc: func(region string, attrs map[string]any) (*pricing.PriceLookup, error) {
				parsed := parseReplicationGroupAttrs(attrs)
				if parsed.NodeType == "" {
					return nil, errors.New("node_type not found")
				}
				runtime := deps.RuntimeOrDefault()
				return runtime.StandardLookupSpec(
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
				).Build(region, attrs)
			},
		},
		Describe: &resourcespec.DescribeSpec{
			BuildFunc: func(price *pricing.Price, attrs map[string]any) map[string]string {
				parsed := parseReplicationGroupAttrs(attrs)
				b := awskit.DescribeBuilder{}
				b.String("node_type", parsed.NodeType)
				if parsed.NumNodeGroupsSet {
					b.Int("node_groups", parsed.NumNodeGroups)
				}
				b.Int("replicas_per_group", parsed.ReplicasPerNodeGroup)
				b.Int("snapshot_retention_days", parsed.SnapshotRetentionDays)
				b.Int("total_nodes", parsed.totalNodes())
				appendNodeCapacityDescribeFields(&b, price)
				return b.Map()
			},
		},
		Standard: &resourcespec.StandardPricingSpec{
			CostFunc: func(price *pricing.Price, index *pricing.PriceIndex, region string, attrs map[string]any) (hourly, monthly float64) {
				parsed := parseReplicationGroupAttrs(attrs)
				totalNodes := parsed.totalNodes()
				_, monthly = handler.ScaledHourlyCost(price.OnDemandUSD, totalNodes)
				monthly += nodeStorageAddOnMonthlyCost(deps.RuntimeOrDefault(), price, index, region, totalNodes, parsed.SnapshotRetentionDays)
				return monthly / handler.HoursPerMonth, monthly
			},
		},
	}
}
