package serverless //nolint:dupl // SQS and SNS are structurally similar usage-based handlers

import (
	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
)

// SQSHandler handles aws_sqs_queue cost estimation
type SQSHandler struct{}

func (h *SQSHandler) Category() handler.CostCategory { return handler.CostCategoryUsageBased }

func (h *SQSHandler) ServiceCode() pricing.ServiceID {
	return awskit.MustService(awskit.ServiceKeySQS)
}

func (h *SQSHandler) BuildLookup(_ string, _ map[string]any) (*pricing.PriceLookup, error) {
	// SQS is usage-based (requests)
	return nil, nil
}

func (h *SQSHandler) Describe(_ *pricing.Price, _ map[string]any) map[string]string { return nil }

func (h *SQSHandler) CalculateCost(_ *pricing.Price, _ *pricing.PriceIndex, _ string, _ map[string]any) (hourly, monthly float64) {
	// SQS: $0.40 per million requests (first million free)
	// Usage-based, no fixed cost
	return 0, 0
}
