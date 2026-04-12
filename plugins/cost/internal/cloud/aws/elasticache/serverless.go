package elasticache

import (
	"fmt"

	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/costutil"
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
				runtime := deps.RuntimeOrDefault()
				return runtime.StandardLookupSpec(
					awskit.ServiceKeyElastiCache,
					"ElastiCache Serverless",
					func(region string, _ map[string]any) (map[string]string, error) {
						prefix := runtime.ResolveUsagePrefix(region)
						return map[string]string{
							"usagetype": prefix + "-ElastiCache:ServerlessStorage",
						}, nil
					},
				).Build(region, nil)
			},
		},
		Describe: &resourcespec.TypedDescribeSpec[serverlessAttrs]{
			BuildFunc: func(_ *pricing.Price, p serverlessAttrs) map[string]string {
				desc := map[string]string{"type": "serverless"}
				if p.Engine != "" {
					desc["engine"] = p.Engine
				}
				if p.StorageMaxGB > 0 {
					desc["storage_max_gb"] = fmt.Sprintf("%.0f", p.StorageMaxGB)
				}
				return desc
			},
		},
		Standard: &resourcespec.TypedStandardPricingSpec[serverlessAttrs]{
			CostFunc: func(price *pricing.Price, _ *pricing.PriceIndex, _ string, p serverlessAttrs) (hourly, monthly float64) {
				if price == nil {
					return 0, 0
				}
				storageGB := p.StorageMaxGB
				if storageGB == 0 {
					storageGB = 1
				}
				costPerGBHour := price.OnDemandUSD
				if costPerGBHour == 0 {
					costPerGBHour = FallbackServerlessStorageCostPerGBHour
				}
				hourly = storageGB * costPerGBHour
				return hourly, hourly * costutil.HoursPerMonth
			},
		},
	}
}
