package elasticache

import (
	"errors"
	"fmt"
	"strings"

	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
)

const (
	// defaultEngine is the Terraform default when engine is not specified.
	defaultEngine = "redis"
	// awsEngineRedis and awsEngineMemcached are the values used in the AWS Pricing API.
	awsEngineRedis     = "Redis"
	awsEngineMemcached = "Memcached"
)

// Backup storage fallback (used when API lookup unavailable).
const FallbackBackupStorageCostPerGBMonth = 0.085

// Data tiering fallback for r6gd/r7gd nodes.
const FallbackDataTieringCostPerGBMonth = 0.0125

// ClusterHandler handles aws_elasticache_cluster cost estimation
type ClusterHandler struct {
	awskit.RuntimeDeps
}

func (h *ClusterHandler) Category() handler.CostCategory { return handler.CostCategoryStandard }

func (h *ClusterHandler) BuildLookup(region string, attrs map[string]any) (*pricing.PriceLookup, error) {
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

	// Use usagetype to select standard on-demand pricing and exclude
	// ExtendedSupport variants which have different rates.
	runtime := h.RuntimeOrDefault()
	spec := runtime.StandardLookupSpec(
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
	)

	return spec.Build(region, attrs)
}

func (h *ClusterHandler) Describe(price *pricing.Price, attrs map[string]any) map[string]string {
	parsed := parseClusterAttrs(attrs)
	b := awskit.DescribeBuilder{}
	b.String("node_type", parsed.NodeType)
	b.String("engine", parsed.Engine)
	b.Int("nodes", parsed.NumCacheNodes)
	b.Int("snapshot_retention_days", parsed.SnapshotRetentionDays)
	if mem := nodeMemoryFromPrice(price); mem > 0 {
		b.String("memory_gib", fmt.Sprintf("%.2f", mem))
	}
	if ssd := nodeSSDFromPrice(price); ssd > 0 {
		b.String("ssd_gib", fmt.Sprintf("%.0f", ssd))
	}
	return b.Map()
}

func (h *ClusterHandler) CalculateCost(price *pricing.Price, index *pricing.PriceIndex, region string, attrs map[string]any) (hourly, monthly float64) {
	if price == nil {
		return 0, 0
	}
	parsed := parseClusterAttrs(attrs)
	numCacheNodes := parsed.NumCacheNodes
	if numCacheNodes == 0 {
		numCacheNodes = 1
	}

	_, monthly = handler.ScaledHourlyCost(price.OnDemandUSD, numCacheNodes)

	// Data tiering cost for nodes with local SSD (r6gd/r7gd)
	monthly += dataTieringCost(h.RuntimeOrDefault(), price, index, region, numCacheNodes)

	// Backup storage cost
	if parsed.SnapshotRetentionDays > 0 {
		monthly += backupStorageCost(h.RuntimeOrDefault(), price, index, region, numCacheNodes, parsed.SnapshotRetentionDays)
	}

	hourly = monthly / handler.HoursPerMonth
	return hourly, monthly
}
