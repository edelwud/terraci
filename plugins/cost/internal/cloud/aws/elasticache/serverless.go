package elasticache

import (
	"fmt"

	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
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

// ServerlessHandler handles aws_elasticache_serverless_cache cost estimation.
// Pricing is based on data storage (GB-hour) and compute (ECPUs).
// Since ECPU usage is unknown at plan time, only storage is estimated
// based on configured cache_usage_limits.
type ServerlessHandler struct {
	awskit.RuntimeDeps
}

func (h *ServerlessHandler) Category() handler.CostCategory { return handler.CostCategoryStandard }

func (h *ServerlessHandler) BuildLookup(region string, _ map[string]any) (*pricing.PriceLookup, error) {
	runtime := h.RuntimeOrDefault()
	spec := runtime.StandardLookupSpec(
		awskit.ServiceKeyElastiCache,
		"ElastiCache Serverless",
		func(region string, _ map[string]any) (map[string]string, error) {
			prefix := runtime.ResolveUsagePrefix(region)
			return map[string]string{
				"usagetype": prefix + "-ElastiCache:ServerlessStorage",
			}, nil
		},
	)

	return spec.Build(region, nil)
}

func (h *ServerlessHandler) Describe(_ *pricing.Price, attrs map[string]any) map[string]string {
	desc := map[string]string{"type": "serverless"}
	if v := handler.GetStringAttr(attrs, "engine"); v != "" {
		desc["engine"] = v
	}
	if storageMax := getServerlessStorageMaxGB(attrs); storageMax > 0 {
		desc["storage_max_gb"] = fmt.Sprintf("%.0f", storageMax)
	}
	return desc
}

func (h *ServerlessHandler) CalculateCost(price *pricing.Price, _ *pricing.PriceIndex, _ string, attrs map[string]any) (hourly, monthly float64) {
	storageGB := getServerlessStorageMaxGB(attrs)
	if storageGB == 0 {
		storageGB = 1 // minimum 1 GB
	}

	// price.OnDemandUSD is per GB-hour from API
	costPerGBHour := price.OnDemandUSD
	if costPerGBHour == 0 {
		costPerGBHour = FallbackServerlessStorageCostPerGBHour
	}

	hourly = storageGB * costPerGBHour
	monthly = hourly * handler.HoursPerMonth
	return hourly, monthly
}

// getServerlessStorageMaxGB extracts maximum storage from cache_usage_limits.
// Terraform schema: cache_usage_limits { data_storage { maximum = N, unit = "GB" } }
func getServerlessStorageMaxGB(attrs map[string]any) float64 {
	limits, ok := attrs["cache_usage_limits"]
	if !ok {
		return 0
	}

	// Terraform plan JSON represents blocks as list of objects
	limitsList, ok := limits.([]any)
	if !ok || len(limitsList) == 0 {
		return 0
	}
	limitsMap, ok := limitsList[0].(map[string]any)
	if !ok {
		return 0
	}

	dataStorage, ok := limitsMap["data_storage"]
	if !ok {
		return 0
	}
	dsList, ok := dataStorage.([]any)
	if !ok || len(dsList) == 0 {
		return 0
	}
	dsMap, ok := dsList[0].(map[string]any)
	if !ok {
		return 0
	}

	return handler.GetFloatAttr(dsMap, "maximum")
}
