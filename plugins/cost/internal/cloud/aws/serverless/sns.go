package serverless

import (
	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/model"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcedef"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcespec"
)

// SNSSpec declares aws_sns_topic cost estimation.
func SNSSpec() resourcespec.TypedSpec[resourcespec.NoAttrs] {
	return resourcespec.TypedSpec[resourcespec.NoAttrs]{
		Type:     resourcedef.ResourceType(awskit.ResourceSNSTopic),
		Category: resourcedef.CostCategoryUsageBased,
		Parse:    resourcespec.ParseNoAttrs,
		Usage: &resourcespec.TypedUsagePricingSpec[resourcespec.NoAttrs]{
			EstimateFunc: func(_ string, _ resourcespec.NoAttrs) model.UsageCostEstimate {
				return model.UsageCostEstimate{Status: model.ResourceEstimateStatusUsageUnknown}
			},
		},
	}
}
