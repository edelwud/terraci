package elasticache

import (
	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"

	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcedef"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcespec"
)

// Serverless pricing fallbacks.
const (
	// FallbackServerlessStorageCostPerGBHour is the per-GB-hour storage cost.
	// AWS charges ~$0.125/GB-month = $0.000171/GB-hour.
	FallbackServerlessStorageCostPerGBHour = 0.000171

	// FallbackServerlessECPUCostPerMillion is per-million ECPU cost.
	// AWS charges $0.0034 per million ECPUs.
	FallbackServerlessECPUCostPerMillion = 0.0034
)

// ServerlessSpec declares aws_elasticache_serverless_cache cost estimation.
func ServerlessSpec(deps awskit.RuntimeDeps) resourcespec.TypedSpec[serverlessAttrs] {
	return resourcespec.TypedSpec[serverlessAttrs]{
		Type:     resourcedef.ResourceType(awskit.ResourceElastiCacheServerlessCache),
		Category: resourcedef.CostCategoryStandard,
		Parse:    parseServerlessAttrs,
		Lookup: &resourcespec.TypedLookupSpec[serverlessAttrs]{
			BuildFunc: func(region string, _ serverlessAttrs) (*pricing.PriceLookup, error) {
				return deps.RuntimeOrDefault().
					NewLookupBuilder(awskit.ServiceKeyElastiCache, "ElastiCache Serverless").
					UsageType(region, "ElastiCache:ServerlessStorage").
					Build(region), nil
			},
		},
		Describe: &resourcespec.TypedDescribeSpec[serverlessAttrs]{
			BuildFunc: func(_ *pricing.Price, p serverlessAttrs) map[string]string {
				return awskit.NewDescribeBuilder().
					String("type", "serverless").
					StringIf(p.Engine != "", "engine", p.Engine).
					FloatIf(p.StorageMaxGB > 0, "storage_max_gb", p.StorageMaxGB, "%.0f").
					Map()
			},
		},
		Standard: &resourcespec.TypedStandardPricingSpec[serverlessAttrs]{
			CostFunc: func(price *pricing.Price, _ *pricing.PriceIndex, _ string, p serverlessAttrs) (hourly, monthly float64) {
				storageGB := p.StorageMaxGB
				if storageGB == 0 {
					storageGB = 1
				}
				return awskit.NewCostBuilder().
					Hourly().
					Scale(storageGB).
					Fallback(FallbackServerlessStorageCostPerGBHour).
					Calc(price, nil, "")
			},
		},
	}
}
