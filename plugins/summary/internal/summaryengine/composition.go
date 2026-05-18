package summaryengine

import (
	"fmt"

	"github.com/edelwud/terraci/pkg/ci"
)

func composeSummaryBody(runtime Runtime, collection *ci.PlanResultCollection, plans []ci.PlanResult, reports []*ci.Report, provider Provider, labels []string) (string, error) {
	body, err := ComposeCommentWithOptions(
		plans,
		reports,
		provider.CommitSHA(),
		provider.PipelineID(),
		collection.GeneratedAt,
		runtime.Config.IncludeDetailsEnabled(),
	)
	if err != nil {
		return "", fmt.Errorf("compose summary comment: %w", err)
	}
	return ci.EmbedManagedLabels(body, labels), nil
}
