package storage //nolint:dupl // SecretsManager and KMS are structurally similar fixed-cost handlers

import (
	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
)

const (
	// SecretsManagerSecretCost is the monthly cost per secret.
	SecretsManagerSecretCost = 0.40
)

// SecretsManagerHandler handles aws_secretsmanager_secret cost estimation
type SecretsManagerHandler struct{}

func (h *SecretsManagerHandler) Category() handler.CostCategory { return handler.CostCategoryFixed }

func (h *SecretsManagerHandler) ServiceCode() pricing.ServiceID {
	return awskit.MustService(awskit.ServiceKeySecretsManager)
}

func (h *SecretsManagerHandler) BuildLookup(_ string, _ map[string]any) (*pricing.PriceLookup, error) {
	return nil, nil
}

func (h *SecretsManagerHandler) Describe(_ *pricing.Price, _ map[string]any) map[string]string {
	return nil
}

func (h *SecretsManagerHandler) CalculateCost(_ *pricing.Price, _ *pricing.PriceIndex, _ string, _ map[string]any) (hourly, monthly float64) {
	// Secrets Manager: $0.40 per secret per month + $0.05 per 10,000 API calls
	return handler.FixedMonthlyCost(SecretsManagerSecretCost)
}
