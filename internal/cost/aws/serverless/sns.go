package serverless //nolint:dupl // SNS and SQS are structurally similar usage-based handlers

import (
	aws "github.com/edelwud/terraci/internal/cost/aws"
	"github.com/edelwud/terraci/internal/cost/pricing"
)

// SNSHandler handles aws_sns_topic cost estimation
type SNSHandler struct{}

func (h *SNSHandler) Category() aws.CostCategory { return aws.CostCategoryUsageBased }

func (h *SNSHandler) ServiceCode() pricing.ServiceCode {
	return pricing.ServiceSNS
}

func (h *SNSHandler) BuildLookup(_ string, _ map[string]any) (*pricing.PriceLookup, error) {
	// SNS is usage-based (publishes + deliveries)
	return nil, nil
}

func (h *SNSHandler) CalculateCost(_ *pricing.Price, _ map[string]any) (hourly, monthly float64) {
	// SNS: $0.50 per million requests
	// Usage-based, no fixed cost
	return 0, 0
}
