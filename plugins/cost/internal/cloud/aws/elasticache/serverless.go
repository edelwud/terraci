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
func ServerlessSpec(deps awskit.RuntimeDeps) resourcespec.ResourceSpec {
	return resourcespec.ResourceSpec{
		Type:     resourcedef.ResourceType(awskit.ResourceElastiCacheServerlessCache),
		Category: resourcedef.CostCategoryStandard,
		Lookup: &resourcespec.LookupSpec{
			BuildFunc: func(region string, _ map[string]any) (*pricing.PriceLookup, error) {
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
		Describe: &resourcespec.DescribeSpec{
			BuildFunc: func(_ *pricing.Price, attrs map[string]any) map[string]string {
				parsed := parseServerlessAttrs(attrs)
				desc := map[string]string{"type": "serverless"}
				if parsed.Engine != "" {
					desc["engine"] = parsed.Engine
				}
				if parsed.StorageMaxGB > 0 {
					desc["storage_max_gb"] = fmt.Sprintf("%.0f", parsed.StorageMaxGB)
				}
				return desc
			},
		},
		Standard: &resourcespec.StandardPricingSpec{
			CostFunc: func(price *pricing.Price, _ *pricing.PriceIndex, _ string, attrs map[string]any) (hourly, monthly float64) {
				if price == nil {
					return 0, 0
				}
				storageGB := parseServerlessAttrs(attrs).StorageMaxGB
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
