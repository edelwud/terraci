package gitlab

import (
	"strings"

	gitlab "gitlab.com/gitlab-org/api/client-go"

	"github.com/edelwud/terraci/internal/ci"
)

// Re-export shared types for backward compatibility
type (
	ModulePlan  = ci.ModulePlan
	PlanStatus  = ci.PlanStatus
	CommentData = ci.CommentData
)

// Re-export shared constants for backward compatibility
const (
	CommentMarker       = ci.CommentMarker
	PlanStatusPending   = ci.PlanStatusPending
	PlanStatusRunning   = ci.PlanStatusRunning
	PlanStatusSuccess   = ci.PlanStatusSuccess
	PlanStatusNoChanges = ci.PlanStatusNoChanges
	PlanStatusChanges   = ci.PlanStatusChanges
	PlanStatusFailed    = ci.PlanStatusFailed
)

// CommentRenderer wraps the shared renderer
type CommentRenderer = ci.CommentRenderer

// NewCommentRenderer creates a new comment renderer
func NewCommentRenderer() *CommentRenderer {
	return ci.NewCommentRenderer()
}

// FindTerraCIComment finds an existing terraci comment in the GitLab notes
func FindTerraCIComment(notes []*gitlab.Note) *gitlab.Note {
	for _, note := range notes {
		if strings.Contains(note.Body, ci.CommentMarker) {
			return note
		}
	}
	return nil
}
