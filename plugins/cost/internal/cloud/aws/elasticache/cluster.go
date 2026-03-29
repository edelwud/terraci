package elasticache

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
)

const defaultEngine = "redis"

// Backup storage fallback (used when API lookup unavailable).
const FallbackBackupStorageCostPerGBMonth = 0.085

// Data tiering fallback for r6gd/r7gd nodes.
const FallbackDataTieringCostPerGBMonth = 0.0125

// ClusterHandler handles aws_elasticache_cluster cost estimation
type ClusterHandler struct{}

func (h *ClusterHandler) Category() handler.CostCategory { return handler.CostCategoryStandard }

func (h *ClusterHandler) ServiceCode() pricing.ServiceCode {
	return pricing.ServiceElastiCache
}

func (h *ClusterHandler) BuildLookup(region string, attrs map[string]any) (*pricing.PriceLookup, error) {
	nodeType := handler.GetStringAttr(attrs, "node_type")
	if nodeType == "" {
		return nil, errors.New("node_type not found")
	}

	engine := handler.GetStringAttr(attrs, "engine")
	if engine == "" {
		engine = defaultEngine
	}

	cacheEngine := "Redis"
	if engine == "memcached" {
		cacheEngine = "Memcached"
	}

	// Use usagetype to select standard on-demand pricing and exclude
	// ExtendedSupport variants which have different rates.
	prefix := awskit.ResolveUsagePrefix(region)
	usagetype := prefix + "-NodeUsage:" + nodeType

	lb := &awskit.LookupBuilder{Service: pricing.ServiceElastiCache, ProductFamily: "Cache Instance"}
	return lb.Build(region, map[string]string{
		"instanceType": nodeType,
		"cacheEngine":  cacheEngine,
		"usagetype":    usagetype,
	}), nil
}

func (h *ClusterHandler) Describe(price *pricing.Price, attrs map[string]any) map[string]string {
	desc := make(map[string]string)
	if v := handler.GetStringAttr(attrs, "node_type"); v != "" {
		desc["node_type"] = v
	}
	if v := handler.GetStringAttr(attrs, "engine"); v != "" {
		desc["engine"] = v
	}
	if v := handler.GetIntAttr(attrs, "num_cache_nodes"); v != 0 {
		desc["nodes"] = strconv.Itoa(v)
	}
	if v := handler.GetIntAttr(attrs, "snapshot_retention_limit"); v > 0 {
		desc["snapshot_retention_days"] = strconv.Itoa(v)
	}
	if mem := nodeMemoryFromPrice(price); mem > 0 {
		desc["memory_gib"] = fmt.Sprintf("%.2f", mem)
	}
	if ssd := nodeSSDFromPrice(price); ssd > 0 {
		desc["ssd_gib"] = fmt.Sprintf("%.0f", ssd)
	}
	return desc
}

func (h *ClusterHandler) CalculateCost(price *pricing.Price, index *pricing.PriceIndex, region string, attrs map[string]any) (hourly, monthly float64) {
	numCacheNodes := handler.GetIntAttr(attrs, "num_cache_nodes")
	if numCacheNodes == 0 {
		numCacheNodes = 1
	}

	_, monthly = handler.ScaledHourlyCost(price.OnDemandUSD, numCacheNodes)

	// Data tiering cost for nodes with local SSD (r6gd/r7gd)
	monthly += dataTieringCost(price, index, region, numCacheNodes)

	// Backup storage cost
	snapshotRetention := handler.GetIntAttr(attrs, "snapshot_retention_limit")
	if snapshotRetention > 0 {
		monthly += backupStorageCost(price, index, region, numCacheNodes, snapshotRetention)
	}

	hourly = monthly / handler.HoursPerMonth
	return hourly, monthly
}
