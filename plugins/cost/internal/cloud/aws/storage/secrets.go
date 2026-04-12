package storage

import (
	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcespec"
)

const (
	// SecretsManagerSecretCost is the monthly cost per secret.
	SecretsManagerSecretCost = 0.40
)

// SecretsManagerSpec declares aws_secretsmanager_secret cost estimation.
func SecretsManagerSpec() resourcespec.ResourceSpec {
	return resourcespec.ResourceSpec{
		Type:     handler.ResourceType(awskit.ResourceSecretsManagerSecret),
		Category: handler.CostCategoryFixed,
		Lookup: &resourcespec.LookupSpec{
			BuildFunc: func(_ string, _ map[string]any) (*pricing.PriceLookup, error) { return nil, nil },
		},
		Describe: &resourcespec.DescribeSpec{
			BuildFunc: func(_ *pricing.Price, _ map[string]any) map[string]string { return nil },
		},
		Fixed: &resourcespec.FixedPricingSpec{
			CostFunc: func(_ string, _ map[string]any) (hourly, monthly float64) {
				return handler.FixedMonthlyCost(SecretsManagerSecretCost)
			},
		},
	}
}
