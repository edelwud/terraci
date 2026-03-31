package pr

import (
	"context"
	"fmt"

	gh "github.com/google/go-github/v68/github"

	"github.com/edelwud/terraci/pkg/ci"
	configpkg "github.com/edelwud/terraci/plugins/github/internal/config"
)

type issueCommentClient interface {
	HasToken() bool
	ListIssueComments(ctx context.Context, prNumber int) ([]*gh.IssueComment, error)
	CreateIssueComment(ctx context.Context, prNumber int, body string) (*gh.IssueComment, error)
	UpdateIssueComment(ctx context.Context, commentID int64, body string) (*gh.IssueComment, error)
}

type Service struct {
	client  issueCommentClient
	config  *configpkg.PRConfig
	context *Context
}

func NewService(cfg *configpkg.PRConfig, client issueCommentClient, ctx *Context) *Service {
	return &Service{
		client:  client,
		config:  cfg,
		context: ctx,
	}
}

func NewServiceFromEnv(cfg *configpkg.PRConfig) *Service {
	return NewService(cfg, NewClientFromEnv(), DetectContext())
}

func (s *Service) IsEnabled() bool {
	if !s.context.InPR {
		return false
	}
	if !s.client.HasToken() {
		return false
	}
	return newCommentPolicy(s.config).enabled()
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
