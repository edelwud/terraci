package storage

import (
	aws "github.com/edelwud/terraci/plugins/cost/engine/aws"
	"github.com/edelwud/terraci/plugins/cost/engine/pricing"
)

// S3Handler handles aws_s3_bucket cost estimation
// Note: S3 is primarily usage-based (storage + requests)
// For fixed cost, we can't estimate without usage data
type S3Handler struct{}

func (h *S3Handler) Category() aws.CostCategory { return aws.CostCategoryUsageBased }

func (h *S3Handler) ServiceCode() pricing.ServiceCode {
	return pricing.ServiceS3
}

func (h *S3Handler) BuildLookup(_ string, _ map[string]any) (*pricing.PriceLookup, error) {
	// S3 bucket itself is free, cost is for storage and requests
	return nil, nil
}

func (h *S3Handler) Describe(_ *pricing.Price, _ map[string]any) map[string]string { return nil }

func (h *S3Handler) CalculateCost(_ *pricing.Price, _ *pricing.PriceIndex, _ string, _ map[string]any) (hourly, monthly float64) {
	// S3: ~$0.023 per GB-month for Standard
	// Without usage data, we can't estimate
	return 0, 0
}
