package storage

import (
	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/model"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcedef"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcespec"
)

// S3Spec declares aws_s3_bucket cost estimation.
func S3Spec() resourcespec.ResourceSpec {
	return resourcespec.ResourceSpec{
		Type:     resourcedef.ResourceType(awskit.ResourceS3Bucket),
		Category: resourcedef.CostCategoryUsageBased,
		Usage: &resourcespec.UsagePricingSpec{
			EstimateFunc: func(_ string, _ map[string]any) model.UsageCostEstimate {
				return model.UsageCostEstimate{Status: model.ResourceEstimateStatusUsageUnknown}
			},
		},
	}
}
