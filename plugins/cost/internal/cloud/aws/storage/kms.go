package storage //nolint:dupl // KMS and SecretsManager are structurally similar fixed-cost handlers

import "github.com/edelwud/terraci/plugins/cost/internal/handler"

const (
	// KMSKeyCost is the monthly cost for a customer-managed KMS key.
	KMSKeyCost = 1.00
)

// KMSHandler handles aws_kms_key cost estimation.
type KMSHandler struct{}

func (h *KMSHandler) Category() handler.CostCategory { return handler.CostCategoryFixed }

func (h *KMSHandler) CalculateFixedCost(_ string, _ map[string]any) (hourly, monthly float64) {
	// KMS: $1.00 per customer-managed key per month
	// AWS managed keys are free
	return handler.FixedMonthlyCost(KMSKeyCost)
}
