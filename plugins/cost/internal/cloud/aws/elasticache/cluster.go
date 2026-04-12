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
func ClusterSpec(deps awskit.RuntimeDeps) resourcespec.ResourceSpec {
	return resourcespec.ResourceSpec{
		Type:     resourcedef.ResourceType(awskit.ResourceElastiCacheCluster),
		Category: resourcedef.CostCategoryStandard,
		Lookup: &resourcespec.LookupSpec{
			BuildFunc: func(region string, attrs map[string]any) (*pricing.PriceLookup, error) {
				parsed := parseClusterAttrs(attrs)
				if parsed.NodeType == "" {
					return nil, errors.New("node_type not found")
				}

				engine := parsed.Engine
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
							"instanceType": parsed.NodeType,
							"cacheEngine":  cacheEngine,
							"usagetype":    prefix + "-NodeUsage:" + parsed.NodeType,
						}, nil
					},
				).Build(region, attrs)
			},
		},
		Describe: &resourcespec.DescribeSpec{
			BuildFunc: func(price *pricing.Price, attrs map[string]any) map[string]string {
				parsed := parseClusterAttrs(attrs)
				b := awskit.DescribeBuilder{}
				b.String("node_type", parsed.NodeType)
				b.String("engine", parsed.Engine)
				b.Int("nodes", parsed.NumCacheNodes)
				b.Int("snapshot_retention_days", parsed.SnapshotRetentionDays)
				appendNodeCapacityDescribeFields(&b, price)
				return b.Map()
			},
		},
		Standard: &resourcespec.StandardPricingSpec{
			CostFunc: func(price *pricing.Price, index *pricing.PriceIndex, region string, attrs map[string]any) (hourly, monthly float64) {
				if price == nil {
					return 0, 0
				}
				parsed := parseClusterAttrs(attrs)
				numCacheNodes := parsed.NumCacheNodes
				if numCacheNodes == 0 {
					numCacheNodes = 1
				}

				_, monthly = costutil.ScaledHourlyCost(price.OnDemandUSD, numCacheNodes)
				monthly += nodeStorageAddOnMonthlyCost(deps.RuntimeOrDefault(), price, index, region, numCacheNodes, parsed.SnapshotRetentionDays)
				return monthly / costutil.HoursPerMonth, monthly
			},
		},
	}
}
