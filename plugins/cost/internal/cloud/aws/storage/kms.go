package storage //nolint:dupl // KMS and SecretsManager are structurally similar fixed-cost handlers

import (
	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
)

const (
	// KMSKeyCost is the monthly cost for a customer-managed KMS key.
	KMSKeyCost = 1.00
)

// KMSHandler handles aws_kms_key cost estimation
type KMSHandler struct{}

func (h *KMSHandler) Category() handler.CostCategory { return handler.CostCategoryFixed }

func (h *KMSHandler) ServiceCode() pricing.ServiceCode {
	return pricing.ServiceKMS
}

func (h *KMSHandler) BuildLookup(_ string, _ map[string]any) (*pricing.PriceLookup, error) {
	return nil, nil
}

func (h *KMSHandler) Describe(_ *pricing.Price, _ map[string]any) map[string]string { return nil }

func (h *KMSHandler) CalculateCost(_ *pricing.Price, _ *pricing.PriceIndex, _ string, _ map[string]any) (hourly, monthly float64) {
	// KMS: $1.00 per customer-managed key per month
	// AWS managed keys are free
	return handler.FixedMonthlyCost(KMSKeyCost)
}
