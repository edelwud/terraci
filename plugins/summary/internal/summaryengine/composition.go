package summaryengine

import (
	"fmt"

	"github.com/edelwud/terraci/pkg/ci"
)

func composeSummaryBody(runtime Runtime, snapshot SummarySnapshot, provider Provider, labels []string) (string, error) {
	collection := snapshot.PlanResults()
	body, err := ComposeCommentWithOptions(
		snapshot,
		CommentMetadata{
			CommitSHA:   provider.CommitSHA(),
			PipelineID:  provider.PipelineID(),
			GeneratedAt: collection.GeneratedAt(),
		},
		CommentOptions{IncludeDetails: runtime.Config.IncludeDetailsEnabled()},
	)
	if err != nil {
		return "", fmt.Errorf("compose summary comment: %w", err)
	}
	return ci.EmbedManagedLabels(body, labels), nil
}
