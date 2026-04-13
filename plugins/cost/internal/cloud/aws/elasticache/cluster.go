package elasticache

import (
	"errors"
	"strings"

	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"

	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcedef"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcespec"
)

const (
	// defaultEngine is the Terraform default when engine is not specified.
	defaultEngine = "redis"
	// awsEngineRedis and awsEngineMemcached are the values used in the AWS Pricing API.
	awsEngineRedis     = "Redis"
	awsEngineMemcached = "Memcached"
)

// ClusterSpec declares aws_elasticache_cluster cost estimation.
func ClusterSpec(deps awskit.RuntimeDeps) resourcespec.TypedSpec[clusterAttrs] {
	return resourcespec.TypedSpec[clusterAttrs]{
		Type:     resourcedef.ResourceType(awskit.ResourceElastiCacheCluster),
		Category: resourcedef.CostCategoryStandard,
		Parse:    parseClusterAttrs,
		Lookup: &resourcespec.TypedLookupSpec[clusterAttrs]{
			BuildFunc: func(region string, p clusterAttrs) (*pricing.PriceLookup, error) {
				if p.NodeType == "" {
					return nil, errors.New("node_type not found")
				}

				engine := p.Engine
				if engine == "" {
					engine = defaultEngine
				}

				return deps.RuntimeOrDefault().
					NewLookupBuilder(awskit.ServiceKeyElastiCache, "Cache Instance").
					Attr("instanceType", p.NodeType).
					AttrMatch("cacheEngine", strings.ToLower(engine), awsEngineRedis, map[string]string{
						"memcached": awsEngineMemcached,
						"redis":     awsEngineRedis,
					}).
					UsageType(region, "NodeUsage:"+p.NodeType).
					Build(region), nil
			},
		},
		Describe: &resourcespec.TypedDescribeSpec[clusterAttrs]{
			BuildFunc: func(price *pricing.Price, p clusterAttrs) map[string]string {
				memoryGiB := 0.0
				ssdGiB := 0.0
				if price != nil {
					memoryGiB = awskit.ParseGiB(price.Attributes["memory"])
					ssdGiB = awskit.ParseGiB(price.Attributes["storage"])
				}
				return awskit.NewDescribeBuilder().
					String("node_type", p.NodeType).
					String("engine", p.Engine).
					Int("nodes", p.NumCacheNodes).
					Int("snapshot_retention_days", p.SnapshotRetentionDays).
					FloatIf(memoryGiB > 0, "memory_gib", memoryGiB, "%.2f").
					FloatIf(ssdGiB > 0, "ssd_gib", ssdGiB, "%.0f").
					Map()
			},
		},
		Standard: &resourcespec.TypedStandardPricingSpec[clusterAttrs]{
			CostFunc: func(price *pricing.Price, index *pricing.PriceIndex, region string, p clusterAttrs) (hourly, monthly float64) {
				rt := deps.RuntimeOrDefault()
				nodes := p.NumCacheNodes
				if nodes == 0 {
					nodes = 1
				}
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
