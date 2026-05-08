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

var _ ci.CommentService = (*Service)(nil)
