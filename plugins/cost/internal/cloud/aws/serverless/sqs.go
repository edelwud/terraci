package serverless

import (
	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/model"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcedef"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcespec"
)

// SQSSpec declares aws_sqs_queue cost estimation.
func SQSSpec() resourcespec.ResourceSpec {
	return resourcespec.ResourceSpec{
		Type:     resourcedef.ResourceType(awskit.ResourceSQSQueue),
		Category: resourcedef.CostCategoryUsageBased,
		Usage: &resourcespec.UsagePricingSpec{
			EstimateFunc: func(_ string, _ map[string]any) model.UsageCostEstimate {
				return model.UsageCostEstimate{Status: model.ResourceEstimateStatusUsageUnknown}
			},
		},
	}
}
