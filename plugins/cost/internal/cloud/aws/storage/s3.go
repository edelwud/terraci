package storage

import (
	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/model"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcedef"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcespec"
)

// S3Spec declares aws_s3_bucket cost estimation.
func S3Spec() resourcespec.TypedSpec[resourcespec.NoAttrs] {
	return resourcespec.TypedSpec[resourcespec.NoAttrs]{
		Type:     resourcedef.ResourceType(awskit.ResourceS3Bucket),
		Category: resourcedef.CostCategoryUsageBased,
		Parse:    resourcespec.ParseNoAttrs,
		Usage: &resourcespec.TypedUsagePricingSpec[resourcespec.NoAttrs]{
			EstimateFunc: func(_ string, _ resourcespec.NoAttrs) model.UsageCostEstimate {
				return model.UsageCostEstimate{Status: model.ResourceEstimateStatusUsageUnknown}
			},
		},
	}
}
