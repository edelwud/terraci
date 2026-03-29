package rds

import (
	"fmt"

	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
)

// ClusterHandler handles aws_rds_cluster cost estimation (Aurora)
type ClusterHandler struct{}

func (h *ClusterHandler) Category() handler.CostCategory { return handler.CostCategoryStandard }

func (h *ClusterHandler) ServiceCode() pricing.ServiceCode {
	return pricing.ServiceRDS
}

func (h *ClusterHandler) BuildLookup(region string, attrs map[string]any) (*pricing.PriceLookup, error) {
	// Aurora cluster itself doesn't have hourly compute cost
	// Cost comes from cluster instances and storage
	// Return a lookup for storage pricing
	engine := handler.GetStringAttr(attrs, "engine")
	if engine == "" {
		engine = DefaultAuroraEngine
	}
	_ = engine // Engine used for validation only

	lb := &awskit.LookupBuilder{Service: pricing.ServiceRDS, ProductFamily: "Database Storage"}
	prefix := awskit.ResolveUsagePrefix(region)
	return lb.Build(region, map[string]string{
		"volumeType": "Aurora:StorageUsage",
		"usagetype":  prefix + "-Aurora:StorageUsage",
	}), nil
}

func (h *ClusterHandler) Describe(_ *pricing.Price, attrs map[string]any) map[string]string {
	d := map[string]string{}
	if v := handler.GetStringAttr(attrs, "engine"); v != "" {
		d["engine"] = v
	}
	if v := handler.GetFloatAttr(attrs, "allocated_storage"); v > 0 {
		d["storage_gb"] = fmt.Sprintf("%.0f", v)
	}
	return d
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
