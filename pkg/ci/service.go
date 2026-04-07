// Package ci defines TerraCI's CI-facing domain: persisted plan/report
// artifacts, review-comment contracts, and provider-shared CI config types.
package ci

import "context"

// CommentService defines the interface for posting plan summaries to PRs/MRs
type CommentService interface {
	IsEnabled() bool
	UpsertComment(ctx context.Context, body string) error
}
