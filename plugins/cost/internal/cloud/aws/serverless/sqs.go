package serverless

import (
	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/model"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcedef"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcespec"
)

// SQSSpec declares aws_sqs_queue cost estimation.
func SQSSpec() resourcespec.TypedSpec[resourcespec.NoAttrs] {
	return resourcespec.TypedSpec[resourcespec.NoAttrs]{
		Type:     resourcedef.ResourceType(awskit.ResourceSQSQueue),
		Category: resourcedef.CostCategoryUsageBased,
		Parse:    resourcespec.ParseNoAttrs,
		Usage: &resourcespec.TypedUsagePricingSpec[resourcespec.NoAttrs]{
			EstimateFunc: func(_ string, _ resourcespec.NoAttrs) model.UsageCostEstimate {
				return model.UsageCostEstimate{Status: model.ResourceEstimateStatusUsageUnknown}
			},
		},
	}
}
