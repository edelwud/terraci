package pr

import (
	"context"
	"fmt"

	gh "github.com/google/go-github/v68/github"

	"github.com/edelwud/terraci/pkg/ci"
)

type issueCommentClient interface {
	HasToken() bool
	ListIssueComments(ctx context.Context, prNumber int) ([]*gh.IssueComment, error)
	CreateIssueComment(ctx context.Context, prNumber int, body string) (*gh.IssueComment, error)
	UpdateIssueComment(ctx context.Context, commentID int64, body string) (*gh.IssueComment, error)
	AddIssueLabels(ctx context.Context, prNumber int, labels []string) error
	RemoveIssueLabel(ctx context.Context, prNumber int, label string) error
}

type Service struct {
	client  issueCommentClient
	context *Context
}

func NewService(client issueCommentClient, ctx *Context) *Service {
	return &Service{
		client:  client,
		context: ctx,
	}
}

func NewServiceFromEnv() *Service {
	return NewService(NewClientFromEnv(), DetectContext())
}

func (s *Service) IsEnabled() bool {
	if !s.context.InPR {
		return false
	}
	if !s.client.HasToken() {
		return false
	}
	return true
}

func (s *Service) UpsertComment(ctx context.Context, body string) error {
	if !s.IsEnabled() {
		return nil
	}

	comments, err := s.client.ListIssueComments(ctx, s.context.PRNumber)
	if err != nil {
		return fmt.Errorf("failed to list PR comments: %w", err)
	}

	existing := FindTerraCIComment(comments)
	if existing != nil {
		if _, err := s.client.UpdateIssueComment(ctx, existing.GetID(), body); err != nil {
			return fmt.Errorf("failed to update PR comment: %w", err)
		}
		return nil
	}

	if _, err := s.client.CreateIssueComment(ctx, s.context.PRNumber, body); err != nil {
		return fmt.Errorf("failed to create PR comment: %w", err)
	}
	return nil
}

// CurrentCommentBody returns the current TerraCI PR comment body, if present.
func (s *Service) CurrentCommentBody(ctx context.Context) (body string, found bool, err error) {
	if !s.IsEnabled() {
		return "", false, nil
	}
	comments, err := s.client.ListIssueComments(ctx, s.context.PRNumber)
	if err != nil {
		return "", false, fmt.Errorf("failed to list PR comments: %w", err)
	}
	existing := FindTerraCIComment(comments)
	if existing == nil {
		return "", false, nil
	}
	return existing.GetBody(), true, nil
}

// SyncLabels synchronizes TerraCI-managed PR labels.
func (s *Service) SyncLabels(ctx context.Context, previous, current []string) error {
	if !s.IsEnabled() {
		return nil
	}
	add, remove := ci.DiffManagedLabels(previous, current)
	for _, label := range remove {
		if err := s.client.RemoveIssueLabel(ctx, s.context.PRNumber, label); err != nil {
			return fmt.Errorf("remove PR label %q: %w", label, err)
		}
	}
	if len(add) > 0 {
		if err := s.client.AddIssueLabels(ctx, s.context.PRNumber, add); err != nil {
			return fmt.Errorf("add PR labels: %w", err)
		}
	}
	return nil
}

var _ ci.CommentService = (*Service)(nil)
var _ ci.ManagedLabelService = (*Service)(nil)
