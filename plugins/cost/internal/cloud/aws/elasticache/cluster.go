package elasticache

import (
	"errors"
	"strings"

	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/costutil"
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
				cacheEngine := awsEngineRedis
				if strings.EqualFold(engine, "memcached") {
					cacheEngine = awsEngineMemcached
				}

				runtime := deps.RuntimeOrDefault()
				return runtime.StandardLookupSpec(
					awskit.ServiceKeyElastiCache,
					"Cache Instance",
					func(region string, _ map[string]any) (map[string]string, error) {
						prefix := runtime.ResolveUsagePrefix(region)
						return map[string]string{
							"instanceType": p.NodeType,
							"cacheEngine":  cacheEngine,
							"usagetype":    prefix + "-NodeUsage:" + p.NodeType,
						}, nil
					},
				).Build(region, nil)
			},
		},
		Describe: &resourcespec.TypedDescribeSpec[clusterAttrs]{
			BuildFunc: func(price *pricing.Price, p clusterAttrs) map[string]string {
				b := awskit.DescribeBuilder{}
				b.String("node_type", p.NodeType)
				b.String("engine", p.Engine)
				b.Int("nodes", p.NumCacheNodes)
				b.Int("snapshot_retention_days", p.SnapshotRetentionDays)
				appendNodeCapacityDescribeFields(&b, price)
				return b.Map()
			},
		},
		Standard: &resourcespec.TypedStandardPricingSpec[clusterAttrs]{
			CostFunc: func(price *pricing.Price, index *pricing.PriceIndex, region string, p clusterAttrs) (hourly, monthly float64) {
				if price == nil {
					return 0, 0
				}
				numCacheNodes := p.NumCacheNodes
				if numCacheNodes == 0 {
					numCacheNodes = 1
				}

				_, monthly = costutil.ScaledHourlyCost(price.OnDemandUSD, numCacheNodes)
				monthly += nodeStorageAddOnMonthlyCost(deps.RuntimeOrDefault(), price, index, region, numCacheNodes, p.SnapshotRetentionDays)
				return monthly / costutil.HoursPerMonth, monthly
			},
		},
	}
}
