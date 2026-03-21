package elasticache

import (
	"fmt"

	aws "github.com/edelwud/terraci/internal/cost/aws"
	"github.com/edelwud/terraci/internal/cost/pricing"
)

// ClusterHandler handles aws_elasticache_cluster cost estimation
type ClusterHandler struct{}

func (h *ClusterHandler) Category() aws.CostCategory { return aws.CostCategoryStandard }

func (h *ClusterHandler) ServiceCode() pricing.ServiceCode {
	return pricing.ServiceElastiCache
}

func (h *ClusterHandler) BuildLookup(region string, attrs map[string]any) (*pricing.PriceLookup, error) {
	nodeType := aws.GetStringAttr(attrs, "node_type")
	if nodeType == "" {
		return nil, fmt.Errorf("node_type not found")
	}

	engine := aws.GetStringAttr(attrs, "engine")
	if engine == "" {
		engine = "redis"
	}

	cacheEngine := "Redis"
	if engine == "memcached" {
		cacheEngine = "Memcached"
	}

	// Use usagetype to select standard on-demand pricing and exclude
	// ExtendedSupport variants which have different rates.
	prefix := aws.ResolveUsagePrefix(region)
	usagetype := prefix + "-NodeUsage:" + nodeType

	lb := &aws.LookupBuilder{Service: pricing.ServiceElastiCache, ProductFamily: "Cache Instance"}
	return lb.Build(region, map[string]string{
		"instanceType": nodeType,
		"cacheEngine":  cacheEngine,
		"usagetype":    usagetype,
	}), nil
}

func (h *ClusterHandler) CalculateCost(price *pricing.Price, attrs map[string]any) (hourly, monthly float64) {
	numCacheNodes := aws.GetIntAttr(attrs, "num_cache_nodes")
	if numCacheNodes == 0 {
		numCacheNodes = 1
	}

	return aws.ScaledHourlyCost(price.OnDemandUSD, numCacheNodes)
}
