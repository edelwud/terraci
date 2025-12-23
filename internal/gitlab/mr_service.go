package gitlab

import (
	"fmt"
	"time"

	"github.com/edelwud/terraci/pkg/config"
)

// MRService handles MR-related operations
type MRService struct {
	client   *Client
	renderer *CommentRenderer
	config   *config.MRConfig
	context  *MRContext
}

// NewMRService creates a new MR service
func NewMRService(cfg *config.MRConfig) *MRService {
	return &MRService{
		client:   NewClientFromEnv(),
		renderer: NewCommentRenderer(),
		config:   cfg,
		context:  DetectMRContext(),
	}
}

// IsEnabled returns true if MR integration is enabled
func (s *MRService) IsEnabled() bool {
	if !s.context.InMR {
		return false
	}

	if !s.client.HasToken() {
		return false
	}

	if s.config == nil {
		return true // Default enabled in MR
	}

	if s.config.Comment == nil {
		return true // Default enabled
	}

	if s.config.Comment.Enabled == nil {
		return true // Default enabled
	}

	return *s.config.Comment.Enabled
}

// UpsertComment creates or updates the terraci comment on the MR
func (s *MRService) UpsertComment(plans []ModulePlan) error {
	if !s.IsEnabled() {
		return nil
	}

	// Check if we should skip comment (on_changes_only)
	if s.config != nil && s.config.Comment != nil && s.config.Comment.OnChangesOnly {
		hasChanges := false
		for i := range plans {
			if plans[i].Status == PlanStatusChanges || plans[i].Status == PlanStatusFailed {
				hasChanges = true
				break
			}
		}
		if !hasChanges {
			return nil
		}
	}

	// Build comment data
	data := &CommentData{
		Plans:       plans,
		CommitSHA:   s.context.CommitSHA,
		PipelineID:  s.context.PipelineID,
		GeneratedAt: time.Now().UTC(),
	}

	// Build pipeline URL
	if s.context.ProjectPath != "" && s.context.PipelineID != "" {
		data.PipelineURL = fmt.Sprintf("%s/%s/-/pipelines/%s",
			s.client.BaseURL(), s.context.ProjectPath, s.context.PipelineID)
	}

	// Render comment
	body := s.renderer.Render(data)

	// Get existing notes to find our comment
	notes, err := s.client.GetMRNotes(s.context.ProjectID, s.context.MRIID)
	if err != nil {
		return fmt.Errorf("failed to get MR notes: %w", err)
	}

	// Find existing terraci comment
	existingNote := FindTerraCIComment(notes)

	if existingNote != nil {
		// Update existing comment
		_, err = s.client.UpdateMRNote(s.context.ProjectID, s.context.MRIID, existingNote.ID, body)
		if err != nil {
			return fmt.Errorf("failed to update MR comment: %w", err)
		}
	} else {
		// Create new comment
		_, err = s.client.CreateMRNote(s.context.ProjectID, s.context.MRIID, body)
		if err != nil {
			return fmt.Errorf("failed to create MR comment: %w", err)
		}
	}

	return nil
}
