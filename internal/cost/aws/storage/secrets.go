package storage //nolint:dupl // SecretsManager and KMS are structurally similar fixed-cost handlers

import (
	aws "github.com/edelwud/terraci/internal/cost/aws"
	"github.com/edelwud/terraci/internal/cost/pricing"
)

const (
	// SecretsManagerSecretCost is the monthly cost per secret.
	SecretsManagerSecretCost = 0.40
)

// SecretsManagerHandler handles aws_secretsmanager_secret cost estimation
type SecretsManagerHandler struct{}

func (h *SecretsManagerHandler) Category() aws.CostCategory { return aws.CostCategoryFixed }

func (h *SecretsManagerHandler) ServiceCode() pricing.ServiceCode {
	return pricing.ServiceSecretsMan
}

func (h *SecretsManagerHandler) BuildLookup(_ string, _ map[string]any) (*pricing.PriceLookup, error) {
	return nil, nil
}

func (h *SecretsManagerHandler) CalculateCost(_ *pricing.Price, _ map[string]any) (hourly, monthly float64) {
	// Secrets Manager: $0.40 per secret per month + $0.05 per 10,000 API calls
	return aws.FixedMonthlyCost(SecretsManagerSecretCost)
}
