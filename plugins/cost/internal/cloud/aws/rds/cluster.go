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

type clusterAttrs struct {
	Engine           string
	AllocatedStorage float64
}

func parseClusterAttrs(attrs map[string]any) clusterAttrs {
	return clusterAttrs{
		Engine:           handler.GetStringAttr(attrs, "engine"),
		AllocatedStorage: handler.GetFloatAttr(attrs, "allocated_storage"),
	}
}

func (h *ClusterHandler) Category() handler.CostCategory { return handler.CostCategoryStandard }

func (h *ClusterHandler) BuildLookup(region string, attrs map[string]any) (*pricing.PriceLookup, error) {
	// Aurora cluster itself doesn't have hourly compute cost
	// Cost comes from cluster instances and storage
	// Return a lookup for storage pricing
	parsed := parseClusterAttrs(attrs)
	engine := parsed.Engine
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
	parsed := parseClusterAttrs(attrs)
	return awskit.NewDescribeBuilder().
		String("engine", parsed.Engine).
		Float("storage_gb", parsed.AllocatedStorage, "%.0f").
		Map()
}

func (h *ClusterHandler) CalculateCost(_ *pricing.Price, _ *pricing.PriceIndex, _ string, attrs map[string]any) (hourly, monthly float64) {
	parsed := parseClusterAttrs(attrs)
	// Aurora storage is billed per GB-month
	// Estimate based on allocated storage or minimum
	allocatedStorage := parsed.AllocatedStorage
	if allocatedStorage == 0 {
		allocatedStorage = 10 // Minimum 10GB
	}

	// Aurora storage: ~$0.10 per GB-month
	monthly = allocatedStorage * AuroraStorageCostPerGB
	hourly = monthly / handler.HoursPerMonth
	return hourly, monthly
}
