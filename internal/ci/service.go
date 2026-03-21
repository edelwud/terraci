// Package ci provides shared CI/CD types and interfaces for provider-agnostic
// plan result handling and PR/MR comment rendering.
package ci

// CommentService defines the interface for posting plan summaries to PRs/MRs
type CommentService interface {
	IsEnabled() bool
	UpsertComment(plans []ModulePlan, policySummary *PolicySummary) error
}
