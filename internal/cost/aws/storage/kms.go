package storage //nolint:dupl // KMS and SecretsManager are structurally similar fixed-cost handlers

import (
	aws "github.com/edelwud/terraci/internal/cost/aws"
	"github.com/edelwud/terraci/internal/cost/pricing"
)

const (
	KMSKeyCost = 1.00
)

// KMSHandler handles aws_kms_key cost estimation
type KMSHandler struct{}

func (h *KMSHandler) Category() aws.CostCategory { return aws.CostCategoryStandard }

func (h *KMSHandler) ServiceCode() pricing.ServiceCode {
	return pricing.ServiceKMS
}

func (h *KMSHandler) BuildLookup(region string, _ map[string]any) (*pricing.PriceLookup, error) {
	lb := &aws.LookupBuilder{Service: pricing.ServiceKMS, ProductFamily: "Key Management Service"}
	return lb.Build(region, nil), nil
}

func (h *KMSHandler) CalculateCost(_ *pricing.Price, _ map[string]any) (hourly, monthly float64) {
	// KMS: $1.00 per key per month (customer managed keys)
	// AWS managed keys are free
	// All key types have the same base cost
	return aws.FixedMonthlyCost(KMSKeyCost)
}
