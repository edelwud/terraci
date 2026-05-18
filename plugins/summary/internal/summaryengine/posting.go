package summaryengine

import (
	"context"
	"fmt"

	log "github.com/caarlos0/log"

	"github.com/edelwud/terraci/pkg/ci"
)

// Provider is the CI-provider surface needed by the summary use case.
type Provider interface {
	CommitSHA() string
	PipelineID() string
	CommentService() (ci.CommentService, bool)
}

// ProviderResolver resolves the active CI provider for the current command.
type ProviderResolver func() (Provider, error)

type summaryPostRequest struct {
	body       string
	labels     []string
	commentSvc ci.CommentService
}

func resolveProvider(resolve ProviderResolver) (Provider, error) {
	if resolve == nil {
		return nil, nil
	}
	return resolve()
}

func prepareSummaryPost(ctx context.Context, req summaryPostRequest) (previousLabels []string, managedSvc ci.ManagedLabelService) {
	if svc, ok := req.commentSvc.(ci.ManagedLabelService); ok {
		managedSvc = svc
		currentBody, found, err := svc.CurrentCommentBody(ctx)
		if err != nil {
			log.Warnf("failed to read current TerraCI comment for managed labels: %v", err)
		} else if found {
			previousLabels = ci.ExtractManagedLabels(currentBody)
		}
	}
	return previousLabels, managedSvc
}

func postSummary(ctx context.Context, req summaryPostRequest, result *Result) error {
	previousLabels, managedSvc := prepareSummaryPost(ctx, req)

	log.Info("updating PR/MR comment")
	if err := req.commentSvc.UpsertComment(ctx, req.body); err != nil {
		return fmt.Errorf("failed to update comment: %w", err)
	}
	result.PostedComment = true
	log.Info("comment updated successfully")

	if managedSvc != nil && (len(previousLabels) > 0 || len(req.labels) > 0) {
		if err := managedSvc.SyncLabels(ctx, previousLabels, req.labels); err != nil {
			log.Warnf("failed to synchronize summary labels: %v", err)
		} else {
			result.SyncedLabels = true
		}
	}

	return nil
}
