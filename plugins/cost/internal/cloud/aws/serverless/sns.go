package serverless //nolint:dupl // SNS and SQS are structurally similar usage-based handlers

import (
	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/model"
)

// SNSHandler handles aws_sns_topic cost estimation.
type SNSHandler struct{}

func (h *SNSHandler) Category() handler.CostCategory { return handler.CostCategoryUsageBased }

func (h *SNSHandler) CalculateUsageCost(_ string, _ map[string]any) model.UsageCostEstimate {
	// SNS: $0.50 per million requests
	// Usage-based, no fixed cost
	return model.UsageCostEstimate{Status: model.ResourceEstimateStatusUsageUnknown}
}
