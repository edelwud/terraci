package storage

import (
	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/model"
)

// S3Handler handles aws_s3_bucket cost estimation
// Note: S3 is primarily usage-based (storage + requests)
// For fixed cost, we can't estimate without usage data.
type S3Handler struct{}

func (h *S3Handler) Category() handler.CostCategory { return handler.CostCategoryUsageBased }

func (h *S3Handler) CalculateUsageCost(_ string, _ map[string]any) model.UsageCostEstimate {
	// S3: ~$0.023 per GB-month for Standard
	// Without usage data, we can't estimate
	return model.UsageCostEstimate{Status: model.ResourceEstimateStatusUsageUnknown}
}
