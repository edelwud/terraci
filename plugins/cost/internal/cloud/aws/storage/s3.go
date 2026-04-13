package storage

import (
	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcedef"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcespec"
)

// S3Spec declares aws_s3_bucket cost estimation.
func S3Spec() resourcespec.TypedSpec[resourcespec.NoAttrs] {
	return resourcespec.UsageUnknownNoAttrsSpec(resourcedef.ResourceType(awskit.ResourceS3Bucket))
}
