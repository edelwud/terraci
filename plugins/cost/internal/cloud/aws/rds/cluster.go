package rds

import (
	"strings"

	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
)

// ClusterHandler handles aws_rds_cluster cost estimation (Aurora).
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

// isAuroraEngine returns true when the engine string is an Aurora variant.
func isAuroraEngine(engine string) bool {
	return strings.HasPrefix(strings.ToLower(engine), "aurora")
}

func (h *ClusterHandler) Category() handler.CostCategory { return handler.CostCategoryStandard }

func (h *ClusterHandler) BuildLookup(region string, attrs map[string]any) (*pricing.PriceLookup, error) {
	// Aurora cluster cost comes from storage; compute cost is on cluster instances.
	parsed := parseClusterAttrs(attrs)
	engine := parsed.Engine
	if engine == "" {
		engine = DefaultAuroraEngine
	}
	if !isAuroraEngine(engine) {
		// Non-Aurora cluster engines are not supported; skip pricing lookup.
		return nil, nil
	}

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

func (h *ClusterHandler) CalculateCost(price *pricing.Price, _ *pricing.PriceIndex, _ string, attrs map[string]any) (hourly, monthly float64) {
	parsed := parseClusterAttrs(attrs)
	allocatedStorage := parsed.AllocatedStorage
	if allocatedStorage == 0 {
		allocatedStorage = 10 // Aurora minimum 10 GB
	}

	// Prefer fetched price; fall back to hardcoded constant.
	costPerGB := AuroraStorageCostPerGB
	if price != nil && price.OnDemandUSD > 0 {
		costPerGB = price.OnDemandUSD
	}

	monthly = allocatedStorage * costPerGB
	hourly = monthly / handler.HoursPerMonth
	return hourly, monthly
}
