package mr

import (
	"context"
	"fmt"

	gitlab "gitlab.com/gitlab-org/api/client-go"

	"github.com/edelwud/terraci/pkg/ci"
	configpkg "github.com/edelwud/terraci/plugins/gitlab/internal/config"
	"github.com/edelwud/terraci/plugins/internal/ciplugin"
)

type noteClient interface {
	HasToken() bool
	GetMRNotes(projectID string, mrIID int64) ([]*gitlab.Note, error)
	CreateMRNote(projectID string, mrIID int64, body string) (*gitlab.Note, error)
	UpdateMRNote(projectID string, mrIID, noteID int64, body string) (*gitlab.Note, error)
}

// Service handles MR-related comment operations.
type Service struct {
	client  noteClient
	config  *configpkg.MRConfig
	context *Context
}

// NewService creates a new MR service with injected dependencies.
func NewService(cfg *configpkg.MRConfig, client noteClient, ctx *Context) *Service {
	return &Service{
		client:  client,
		config:  cfg,
		context: ctx,
	}
}

// NewServiceFromEnv creates a new MR service with dependencies from environment.
func NewServiceFromEnv(cfg *configpkg.MRConfig) *Service {
	return NewService(cfg, NewClientFromEnv(), DetectContext())
}

// IsEnabled returns true if MR integration is enabled.
func (s *Service) IsEnabled() bool {
	if !s.context.InMR {
		return false
	}
	if !s.client.HasToken() {
		return false
	}
	return ciplugin.CommentEnabled(s.config)
}

// UpsertComment creates or updates the terraci comment on the MR.
func (s *Service) UpsertComment(_ context.Context, body string) error {
	if !s.IsEnabled() {
		return nil
	}

	notes, err := s.client.GetMRNotes(s.context.ProjectID, s.context.MRIID)
	if err != nil {
		return fmt.Errorf("failed to get MR notes: %w", err)
	}

	existingNote := FindTerraCIComment(notes)
	if existingNote != nil {
		if _, err = s.client.UpdateMRNote(s.context.ProjectID, s.context.MRIID, existingNote.ID, body); err != nil {
			return fmt.Errorf("failed to update MR comment: %w", err)
		}
		return nil
	}

	if _, err = s.client.CreateMRNote(s.context.ProjectID, s.context.MRIID, body); err != nil {
		return fmt.Errorf("failed to create MR comment: %w", err)
	}

	return nil
}

// Ensure Service satisfies CommentService at compile time.
var _ ci.CommentService = (*Service)(nil)
