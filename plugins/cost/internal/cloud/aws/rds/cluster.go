package rds

import (
	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
)

// ClusterHandler handles aws_rds_cluster cost estimation (Aurora)
type ClusterHandler struct {
	awskit.RuntimeDeps
}

func (h *ClusterHandler) Category() handler.CostCategory { return handler.CostCategoryStandard }

func (h *ClusterHandler) BuildLookup(region string, attrs map[string]any) (*pricing.PriceLookup, error) {
	// Aurora cluster itself doesn't have hourly compute cost
	// Cost comes from cluster instances and storage
	// Return a lookup for storage pricing
	engine := handler.GetStringAttr(attrs, "engine")
	if engine == "" {
		engine = DefaultAuroraEngine
	}
	_ = engine // Engine used for validation only

	runtime := h.RuntimeOrDefault()
	spec := runtime.StandardLookupSpec(
		awskit.ServiceKeyRDS,
		"Database Storage",
		func(region string, _ map[string]any) (map[string]string, error) {
			prefix := runtime.ResolveUsagePrefix(region)
			return map[string]string{
				"volumeType": "Aurora:StorageUsage",
				"usagetype":  prefix + "-Aurora:StorageUsage",
			}, nil
		},
	)

	return spec.Build(region, attrs)
}

func (h *ClusterHandler) Describe(_ *pricing.Price, attrs map[string]any) map[string]string {
	return awskit.NewDescribeBuilder().
		String("engine", handler.GetStringAttr(attrs, "engine")).
		Float("storage_gb", handler.GetFloatAttr(attrs, "allocated_storage"), "%.0f").
		Map()
}

func (h *ClusterHandler) CalculateCost(_ *pricing.Price, _ *pricing.PriceIndex, _ string, attrs map[string]any) (hourly, monthly float64) {
	// Aurora storage is billed per GB-month
	// Estimate based on allocated storage or minimum
	allocatedStorage := handler.GetFloatAttr(attrs, "allocated_storage")
	if allocatedStorage == 0 {
		allocatedStorage = 10 // Minimum 10GB
	}

	// Aurora storage: ~$0.10 per GB-month
	monthly = allocatedStorage * AuroraStorageCostPerGB
	hourly = monthly / handler.HoursPerMonth
	return hourly, monthly
}
