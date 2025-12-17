package gitlab

import (
	"strings"
	"testing"
	"time"
)

func TestCommentRenderer_Render(t *testing.T) {
	renderer := NewCommentRenderer()

	plans := []ModulePlan{
		{
			ModuleID:    "platform/stage/eu-central-1/vpc",
			Service:     "platform",
			Environment: "stage",
			Region:      "eu-central-1",
			Module:      "vpc",
			Status:      PlanStatusNoChanges,
			Summary:     "No changes. Infrastructure is up-to-date.",
		},
		{
			ModuleID:    "platform/stage/eu-central-1/eks",
			Service:     "platform",
			Environment: "stage",
			Region:      "eu-central-1",
			Module:      "eks",
			Status:      PlanStatusChanges,
			Summary:     "Plan: 2 to add, 1 to change, 0 to destroy.",
			Details:     "# Some terraform plan output here",
		},
		{
			ModuleID:    "platform/prod/eu-central-1/vpc",
			Service:     "platform",
			Environment: "prod",
			Region:      "eu-central-1",
			Module:      "vpc",
			Status:      PlanStatusFailed,
			Error:       "Error acquiring state lock",
		},
	}

	data := &CommentData{
		Plans:       plans,
		CommitSHA:   "abc123def456",
		PipelineID:  "12345",
		PipelineURL: "https://gitlab.com/group/project/-/pipelines/12345",
		GeneratedAt: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
	}

	result := renderer.Render(data)

	// Check marker
	if !strings.Contains(result, CommentMarker) {
		t.Error("missing comment marker")
	}

	// Check header
	if !strings.Contains(result, "## 🏗️ Terraform Plan Summary") {
		t.Error("missing header")
	}

	// Check stats
	if !strings.Contains(result, "**3** modules") {
		t.Error("missing total modules count")
	}

	// Check environment sections
	if !strings.Contains(result, "### 📦 Environment: `stage`") {
		t.Error("missing stage environment section")
	}
	if !strings.Contains(result, "### 📦 Environment: `prod`") {
		t.Error("missing prod environment section")
	}

	// Check module IDs in table
	if !strings.Contains(result, "`platform/stage/eu-central-1/vpc`") {
		t.Error("missing vpc module in table")
	}

	// Check status icons
	if !strings.Contains(result, "| ✅ |") {
		t.Error("missing success status icon")
	}
	if !strings.Contains(result, "| 🔄 |") {
		t.Error("missing changes status icon")
	}
	if !strings.Contains(result, "| ❌ |") {
		t.Error("missing failed status icon")
	}

	// Check expandable details
	if !strings.Contains(result, "<details>") {
		t.Error("missing expandable details")
	}

	// Check footer
	if !strings.Contains(result, "terraci") {
		t.Error("missing terraci reference in footer")
	}
	if !strings.Contains(result, "Pipeline #12345") {
		t.Error("missing pipeline link")
	}
	if !strings.Contains(result, "abc123de") {
		t.Error("missing commit SHA")
	}
}

func TestCommentRenderer_StatusIcon(t *testing.T) {
	renderer := NewCommentRenderer()

	tests := []struct {
		status   PlanStatus
		expected string
	}{
		{PlanStatusSuccess, "✅"},
		{PlanStatusNoChanges, "✅"},
		{PlanStatusChanges, "🔄"},
		{PlanStatusFailed, "❌"},
		{PlanStatusPending, "⏳"},
		{PlanStatusRunning, "🔄"},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			result := renderer.statusIcon(tt.status)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestFindTerraCIComment(t *testing.T) {
	notes := []Note{
		{ID: 1, Body: "Some other comment"},
		{ID: 2, Body: CommentMarker + "\n\n## Terraform Plan"},
		{ID: 3, Body: "Another comment"},
	}

	found := FindTerraCIComment(notes)
	if found == nil {
		t.Fatal("expected to find terraci comment")
	}
	if found.ID != 2 {
		t.Errorf("expected note ID 2, got %d", found.ID)
	}
}

func TestFindTerraCIComment_NotFound(t *testing.T) {
	notes := []Note{
		{ID: 1, Body: "Some comment"},
		{ID: 2, Body: "Another comment"},
	}

	found := FindTerraCIComment(notes)
	if found != nil {
		t.Error("expected nil, found a comment")
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"exactly10!", 10, "exactly10!"},
		{"this is a long string", 10, "this is..."},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := truncate(tt.input, tt.maxLen)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}
