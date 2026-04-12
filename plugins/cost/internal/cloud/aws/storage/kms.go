package storage

import (
	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/costutil"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcedef"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcespec"
)

const (
	// KMSKeyCost is the monthly cost for a customer-managed KMS key.
	KMSKeyCost = 1.00
)

// KMSSpec declares aws_kms_key cost estimation.
func KMSSpec() resourcespec.TypedSpec[resourcespec.NoAttrs] {
	return resourcespec.TypedSpec[resourcespec.NoAttrs]{
		Type:     resourcedef.ResourceType(awskit.ResourceKMSKey),
		Category: resourcedef.CostCategoryFixed,
		Parse:    resourcespec.ParseNoAttrs,
		Fixed: &resourcespec.TypedFixedPricingSpec[resourcespec.NoAttrs]{
			CostFunc: func(_ string, _ resourcespec.NoAttrs) (hourly, monthly float64) {
				return costutil.FixedMonthlyCost(KMSKeyCost)
			},
		},
	}
}
