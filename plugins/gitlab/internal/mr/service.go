package mr

import (
	"context"
	"fmt"

	gitlab "gitlab.com/gitlab-org/api/client-go"

	"github.com/edelwud/terraci/pkg/ci"
)

type noteClient interface {
	HasToken() bool
	GetMRNotes(projectID string, mrIID int64) ([]*gitlab.Note, error)
	CreateMRNote(projectID string, mrIID int64, body string) (*gitlab.Note, error)
	UpdateMRNote(projectID string, mrIID, noteID int64, body string) (*gitlab.Note, error)
	AddMRLabels(projectID string, mrIID int64, labels []string) error
	RemoveMRLabels(projectID string, mrIID int64, labels []string) error
}

// Service handles MR-related comment operations.
type Service struct {
	client  noteClient
	context *Context
}

// NewService creates a new MR service with injected dependencies.
func NewService(client noteClient, ctx *Context) *Service {
	return &Service{
		client:  client,
		context: ctx,
	}
}

// NewServiceFromEnv creates a new MR service with dependencies from environment.
func NewServiceFromEnv() *Service {
	return NewService(NewClientFromEnv(), DetectContext())
}

// IsEnabled returns true if MR integration is enabled.
func (s *Service) IsEnabled() bool {
	if !s.context.InMR {
		return false
	}
	if !s.client.HasToken() {
		return false
	}
	return true
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

// CurrentCommentBody returns the current TerraCI MR comment body, if present.
func (s *Service) CurrentCommentBody(_ context.Context) (body string, found bool, err error) {
	if !s.IsEnabled() {
		return "", false, nil
	}
	notes, err := s.client.GetMRNotes(s.context.ProjectID, s.context.MRIID)
	if err != nil {
		return "", false, fmt.Errorf("failed to get MR notes: %w", err)
	}
	existingNote := FindTerraCIComment(notes)
	if existingNote == nil {
		return "", false, nil
	}
	return existingNote.Body, true, nil
}

// SyncLabels synchronizes TerraCI-managed MR labels.
func (s *Service) SyncLabels(_ context.Context, previous, current []string) error {
	if !s.IsEnabled() {
		return nil
	}
	add, remove := ci.DiffManagedLabels(previous, current)
	if len(remove) > 0 {
		if err := s.client.RemoveMRLabels(s.context.ProjectID, s.context.MRIID, remove); err != nil {
			return fmt.Errorf("remove MR labels: %w", err)
		}
	}
	if len(add) > 0 {
		if err := s.client.AddMRLabels(s.context.ProjectID, s.context.MRIID, add); err != nil {
			return fmt.Errorf("add MR labels: %w", err)
		}
	}
	return nil
}

// Ensure Service satisfies CommentService at compile time.
var _ ci.CommentService = (*Service)(nil)
var _ ci.ManagedLabelService = (*Service)(nil)
