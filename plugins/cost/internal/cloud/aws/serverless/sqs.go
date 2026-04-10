package serverless //nolint:dupl // SQS and SNS are structurally similar usage-based handlers

import (
	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/model"
)

// SQSHandler handles aws_sqs_queue cost estimation.
type SQSHandler struct{}

func (h *SQSHandler) Category() handler.CostCategory { return handler.CostCategoryUsageBased }

func (h *SQSHandler) CalculateUsageCost(_ string, _ map[string]any) model.UsageCostEstimate {
	// SQS: $0.40 per million requests (first million free)
	// Usage-based, no fixed cost
	return model.UsageCostEstimate{Status: model.ResourceEstimateStatusUsageUnknown}
}
