package elasticache

import (
	"errors"

	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/costutil"
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
				return runtime.StandardLookupSpec(
					awskit.ServiceKeyElastiCache,
					"Cache Instance",
					func(region string, _ map[string]any) (map[string]string, error) {
						prefix := runtime.ResolveUsagePrefix(region)
						return map[string]string{
							"instanceType": p.NodeType,
							"cacheEngine":  "Redis",
							"usagetype":    prefix + "-NodeUsage:" + p.NodeType,
						}, nil
					},
				).Build(region, nil)
			},
		},
		Describe: &resourcespec.TypedDescribeSpec[replicationGroupAttrs]{
			BuildFunc: func(price *pricing.Price, p replicationGroupAttrs) map[string]string {
				b := awskit.DescribeBuilder{}
				b.String("node_type", p.NodeType)
				if p.NumNodeGroupsSet {
					b.Int("node_groups", p.NumNodeGroups)
				}
				b.Int("replicas_per_group", p.ReplicasPerNodeGroup)
				b.Int("snapshot_retention_days", p.SnapshotRetentionDays)
				b.Int("total_nodes", p.totalNodes())
				appendNodeCapacityDescribeFields(&b, price)
				return b.Map()
			},
		},
		Standard: &resourcespec.TypedStandardPricingSpec[replicationGroupAttrs]{
			CostFunc: func(price *pricing.Price, index *pricing.PriceIndex, region string, p replicationGroupAttrs) (hourly, monthly float64) {
				totalNodes := p.totalNodes()
				_, monthly = costutil.ScaledHourlyCost(price.OnDemandUSD, totalNodes)
				monthly += nodeStorageAddOnMonthlyCost(deps.RuntimeOrDefault(), price, index, region, totalNodes, p.SnapshotRetentionDays)
				return monthly / costutil.HoursPerMonth, monthly
			},
		},
	}
}
