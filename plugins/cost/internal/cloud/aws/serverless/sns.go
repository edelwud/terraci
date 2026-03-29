package serverless //nolint:dupl // SNS and SQS are structurally similar usage-based handlers

import (
	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
)

// SNSHandler handles aws_sns_topic cost estimation.
type SNSHandler struct{}

func (h *SNSHandler) Category() handler.CostCategory { return handler.CostCategoryUsageBased }

func (h *SNSHandler) CalculateCost(_ *pricing.Price, _ *pricing.PriceIndex, _ string, _ map[string]any) (hourly, monthly float64) {
	// SNS: $0.50 per million requests
	// Usage-based, no fixed cost
	return 0, 0
}
