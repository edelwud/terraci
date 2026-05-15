// Package ci defines TerraCI's CI-facing domain: persisted plan/report
// artifacts, review-comment contracts, and provider-shared CI config types.
package ci

import "context"

// CommentService defines the interface for posting plan summaries to PRs/MRs
type CommentService interface {
	IsEnabled() bool
	UpsertComment(ctx context.Context, body string) error
}

// ManagedLabelService is an optional extension for comment services that can
// synchronize labels owned by the TerraCI summary comment.
type ManagedLabelService interface {
	CurrentCommentBody(ctx context.Context) (body string, found bool, err error)
	SyncLabels(ctx context.Context, previous, current []string) error
}
