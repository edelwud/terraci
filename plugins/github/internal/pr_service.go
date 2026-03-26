package githubci

import (
	"context"
	"fmt"
	"strings"

	"github.com/edelwud/terraci/pkg/ci"
)

// PRService handles GitHub PR comment operations
type PRService struct {
	client  *Client
	config  *PRConfig
	context *PRContext
}

// NewPRService creates a new PR service with injected dependencies.
func NewPRService(cfg *PRConfig, client *Client, ctx *PRContext) *PRService {
	return &PRService{
		client:  client,
		config:  cfg,
		context: ctx,
	}
}

// NewPRServiceFromEnv creates a new PR service with dependencies from environment.
func NewPRServiceFromEnv(cfg *PRConfig) *PRService {
	return NewPRService(cfg, NewClientFromEnv(), DetectPRContext())
}

// IsEnabled returns true if PR integration is enabled
func (s *PRService) IsEnabled() bool {
	if !s.context.InPR {
		return false
	}

	if !s.client.HasToken() {
		return false
	}

	if s.config == nil {
		return true
	}

	if s.config.Comment == nil {
		return true
	}

	if s.config.Comment.Enabled == nil {
		return true
	}

	return *s.config.Comment.Enabled
}

// UpsertComment creates or updates the terraci comment on the PR
func (s *PRService) UpsertComment(body string) error {
	if !s.IsEnabled() {
		return nil
	}

	ctx := context.Background()

	// Find existing terraci comment
	comments, err := s.client.ListIssueComments(ctx, s.context.PRNumber)
	if err != nil {
		return fmt.Errorf("failed to list PR comments: %w", err)
	}

	var existingID int64
	for _, comment := range comments {
		if comment.Body != nil && strings.Contains(*comment.Body, ci.CommentMarker) {
			existingID = comment.GetID()
			break
		}
	}

	if existingID != 0 {
		_, err = s.client.UpdateIssueComment(ctx, existingID, body)
		if err != nil {
			return fmt.Errorf("failed to update PR comment: %w", err)
		}
	} else {
		_, err = s.client.CreateIssueComment(ctx, s.context.PRNumber, body)
		if err != nil {
			return fmt.Errorf("failed to create PR comment: %w", err)
		}
	}

	return nil
}

// Ensure PRService satisfies CommentService at compile time.
var _ ci.CommentService = (*PRService)(nil)
