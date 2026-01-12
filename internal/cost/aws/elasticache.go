package aws

import (
	"fmt"

	"github.com/edelwud/terraci/internal/cost/pricing"
)

// ElastiCacheClusterHandler handles aws_elasticache_cluster cost estimation
type ElastiCacheClusterHandler struct{}

func (h *ElastiCacheClusterHandler) ServiceCode() pricing.ServiceCode {
	return pricing.ServiceElastiCache
}

func (h *ElastiCacheClusterHandler) BuildLookup(region string, attrs map[string]interface{}) (*pricing.PriceLookup, error) {
	nodeType := getStringAttr(attrs, "node_type")
	if nodeType == "" {
		return nil, fmt.Errorf("node_type not found")
	}

	engine := getStringAttr(attrs, "engine")
	if engine == "" {
		engine = "redis"
	}

	regionName := pricing.RegionMapping[region]
	if regionName == "" {
		regionName = region
	}

	// Cache engine naming
	cacheEngine := "Redis"
	if engine == "memcached" {
		cacheEngine = "Memcached"
	}

	return &pricing.PriceLookup{
		ServiceCode:   pricing.ServiceElastiCache,
		Region:        region,
		ProductFamily: "Cache Instance",
		Attributes: map[string]string{
			"instanceType": nodeType,
			"location":     regionName,
			"cacheEngine":  cacheEngine,
		},
	}, nil
}

func (h *ElastiCacheClusterHandler) CalculateCost(price *pricing.Price, attrs map[string]interface{}) (hourly, monthly float64) {
	numCacheNodes := getIntAttr(attrs, "num_cache_nodes")
	if numCacheNodes == 0 {
		numCacheNodes = 1
	}

	hourly = price.OnDemandUSD * float64(numCacheNodes)
	monthly = hourly * HoursPerMonth
	return hourly, monthly
}

// ElastiCacheReplicationGroupHandler handles aws_elasticache_replication_group cost estimation
type ElastiCacheReplicationGroupHandler struct{}

func (h *ElastiCacheReplicationGroupHandler) ServiceCode() pricing.ServiceCode {
	return pricing.ServiceElastiCache
}

func (h *ElastiCacheReplicationGroupHandler) BuildLookup(region string, attrs map[string]interface{}) (*pricing.PriceLookup, error) {
	nodeType := getStringAttr(attrs, "node_type")
	if nodeType == "" {
		return nil, fmt.Errorf("node_type not found")
	}

	regionName := pricing.RegionMapping[region]
	if regionName == "" {
		regionName = region
	}

	return &pricing.PriceLookup{
		ServiceCode:   pricing.ServiceElastiCache,
		Region:        region,
		ProductFamily: "Cache Instance",
		Attributes: map[string]string{
			"instanceType": nodeType,
			"location":     regionName,
			"cacheEngine":  "Redis",
		},
	}, nil
}

func (h *ElastiCacheReplicationGroupHandler) CalculateCost(price *pricing.Price, attrs map[string]interface{}) (hourly, monthly float64) {
	// Calculate total nodes: num_node_groups * replicas_per_node_group
	numNodeGroups := getIntAttr(attrs, "num_node_groups")
	if numNodeGroups == 0 {
		numNodeGroups = 1
	}

	replicasPerGroup := getIntAttr(attrs, "replicas_per_node_group")
	// Total nodes = primary (1 per group) + replicas
	totalNodes := numNodeGroups * (1 + replicasPerGroup)

	// Legacy attribute support
	if totalNodes == 1 {
		numberCacheClusters := getIntAttr(attrs, "number_cache_clusters")
		if numberCacheClusters > 0 {
			totalNodes = numberCacheClusters
		}
	}

	hourly = price.OnDemandUSD * float64(totalNodes)
	monthly = hourly * HoursPerMonth
	return hourly, monthly
}
