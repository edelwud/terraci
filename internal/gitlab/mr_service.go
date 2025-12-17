package gitlab

import (
	"fmt"
	"strings"
	"time"

	"github.com/edelwud/terraci/internal/discovery"
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

// IsInMR returns true if we're running in a GitLab MR pipeline
func (s *MRService) IsInMR() bool {
	return s.context.InMR
}

// GetContext returns the MR context
func (s *MRService) GetContext() *MRContext {
	return s.context
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
			s.client.baseURL, s.context.ProjectPath, s.context.PipelineID)
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

// AddLabels adds labels to the MR based on the affected modules
func (s *MRService) AddLabels(modules []*discovery.Module) error {
	if !s.context.InMR || !s.client.HasToken() {
		return nil
	}

	if s.config == nil || len(s.config.Labels) == 0 {
		return nil
	}

	// Expand label placeholders and collect unique labels
	labelSet := make(map[string]bool)

	for _, labelTemplate := range s.config.Labels {
		// If template has no placeholders, add as-is
		if !strings.Contains(labelTemplate, "{") {
			labelSet[labelTemplate] = true
			continue
		}

		// Expand for each module
		for _, m := range modules {
			label := expandLabelPlaceholders(labelTemplate, m)
			if label != "" {
				labelSet[label] = true
			}
		}
	}

	if len(labelSet) == 0 {
		return nil
	}

	// Convert to slice
	labels := make([]string, 0, len(labelSet))
	for label := range labelSet {
		labels = append(labels, label)
	}

	return s.client.AddMRLabels(s.context.ProjectID, s.context.MRIID, labels)
}

// expandLabelPlaceholders expands placeholders in a label template
func expandLabelPlaceholders(template string, m *discovery.Module) string {
	result := template

	result = strings.ReplaceAll(result, "{service}", m.Service)
	result = strings.ReplaceAll(result, "{environment}", m.Environment)
	result = strings.ReplaceAll(result, "{env}", m.Environment)
	result = strings.ReplaceAll(result, "{region}", m.Region)
	result = strings.ReplaceAll(result, "{module}", m.Module)

	if m.Submodule != "" {
		result = strings.ReplaceAll(result, "{submodule}", m.Submodule)
	} else {
		// Remove {submodule} if not present
		result = strings.ReplaceAll(result, "{submodule}", "")
		result = strings.ReplaceAll(result, "//", "/") // Clean up double slashes
		result = strings.TrimSuffix(result, "/")
	}

	return result
}

// ModulesToPlans converts discovery modules to plan structures (for initial state)
func ModulesToPlans(modules []*discovery.Module) []ModulePlan {
	plans := make([]ModulePlan, len(modules))
	for i, m := range modules {
		plans[i] = ModulePlan{
			ModuleID:    m.ID(),
			ModulePath:  m.RelativePath,
			Service:     m.Service,
			Environment: m.Environment,
			Region:      m.Region,
			Module:      m.Module,
			Status:      PlanStatusPending,
		}
	}
	return plans
}
