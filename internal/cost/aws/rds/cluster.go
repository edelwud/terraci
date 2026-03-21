package rds

import (
	aws "github.com/edelwud/terraci/internal/cost/aws"
	"github.com/edelwud/terraci/internal/cost/pricing"
)

// ClusterHandler handles aws_rds_cluster cost estimation (Aurora)
type ClusterHandler struct{}

func (h *ClusterHandler) Category() aws.CostCategory { return aws.CostCategoryStandard }

func (h *ClusterHandler) ServiceCode() pricing.ServiceCode {
	return pricing.ServiceRDS
}

func (h *ClusterHandler) BuildLookup(region string, attrs map[string]any) (*pricing.PriceLookup, error) {
	// Aurora cluster itself doesn't have hourly compute cost
	// Cost comes from cluster instances and storage
	// Return a lookup for storage pricing
	engine := aws.GetStringAttr(attrs, "engine")
	if engine == "" {
		engine = DefaultAuroraEngine
	}
	_ = engine // Engine used for validation only

	lb := &aws.LookupBuilder{Service: pricing.ServiceRDS, ProductFamily: "Database Storage"}
	return lb.Build(region, map[string]string{
		"volumeType": "Aurora:StorageUsage",
		"usagetype":  region + "-Aurora:StorageUsage",
	}), nil
}

func (h *ClusterHandler) CalculateCost(_ *pricing.Price, attrs map[string]any) (hourly, monthly float64) {
	// Aurora storage is billed per GB-month
	// Estimate based on allocated storage or minimum
	allocatedStorage := aws.GetFloatAttr(attrs, "allocated_storage")
	if allocatedStorage == 0 {
		allocatedStorage = 10 // Minimum 10GB
	}

	// Aurora storage: ~$0.10 per GB-month
	monthly = allocatedStorage * AuroraStorageCostPerGB
	hourly = monthly / aws.HoursPerMonth
	return hourly, monthly
}
