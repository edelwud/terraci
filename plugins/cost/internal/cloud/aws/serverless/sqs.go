package serverless

import (
	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcedef"
	"github.com/edelwud/terraci/plugins/cost/internal/resourcespec"
)

// SQSSpec declares aws_sqs_queue cost estimation.
func SQSSpec() resourcespec.TypedSpec[resourcespec.NoAttrs] {
	return resourcespec.UsageUnknownNoAttrsSpec(resourcedef.ResourceType(awskit.ResourceSQSQueue))
}
