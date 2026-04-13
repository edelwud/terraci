package serverless

import (
	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcedef"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcespec"
)

// SNSSpec declares aws_sns_topic cost estimation.
func SNSSpec() resourcespec.TypedSpec[resourcespec.NoAttrs] {
	return resourcespec.UsageUnknownNoAttrsSpec(resourcedef.ResourceType(awskit.ResourceSNSTopic))
}
