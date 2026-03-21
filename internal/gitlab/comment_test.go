package gitlab

import (
	"strings"
	"testing"
	"time"

	gitlab "gitlab.com/gitlab-org/api/client-go"

	"github.com/edelwud/terraci/internal/ci"
)

func TestCommentRenderer_Render(t *testing.T) {
	renderer := ci.NewCommentRenderer()

	plans := []ci.ModulePlan{
		{
			ModuleID:   "platform/stage/eu-central-1/vpc",
			Components: map[string]string{"service": "platform", "environment": "stage", "region": "eu-central-1", "module": "vpc"},
			Status:     ci.PlanStatusNoChanges,
			Summary:    "No changes. Infrastructure is up-to-date.",
		},
		{
			ModuleID:          "platform/stage/eu-central-1/eks",
			Components:        map[string]string{"service": "platform", "environment": "stage", "region": "eu-central-1", "module": "eks"},
			Status:            ci.PlanStatusChanges,
			Summary:           "+2 ~1",
			StructuredDetails: "**Create:**\n- `aws_instance.web`\n- `aws_instance.api`\n\n**Update:**\n- `aws_security_group.main`",
			RawPlanOutput:     "# Some terraform plan output here",
		},
		{
			ModuleID:   "platform/prod/eu-central-1/vpc",
			Components: map[string]string{"service": "platform", "environment": "prod", "region": "eu-central-1", "module": "vpc"},
			Status:     ci.PlanStatusFailed,
			Error:      "Error acquiring state lock",
		},
	}

	data := &ci.CommentData{
		Plans:       plans,
		CommitSHA:   "abc123def456",
		PipelineID:  "12345",
		PipelineURL: "https://gitlab.com/group/project/-/pipelines/12345",
		GeneratedAt: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
	}

	result := renderer.Render(data)

	if !strings.Contains(result, ci.CommentMarker) {
		t.Error("missing comment marker")
	}

	if !strings.Contains(result, "## 🏗️ Terraform Plan Summary") {
		t.Error("missing header")
	}

	if !strings.Contains(result, "**3** modules") {
		t.Error("missing total modules count")
	}

	if !strings.Contains(result, "### 📦 Environment: `stage`") {
		t.Error("missing stage environment section")
	}
	if !strings.Contains(result, "### 📦 Environment: `prod`") {
		t.Error("missing prod environment section")
	}

	if !strings.Contains(result, "`platform/stage/eu-central-1/vpc`") {
		t.Error("missing vpc module in table")
	}

	if !strings.Contains(result, "| ✅ |") {
		t.Error("missing success status icon")
	}
	if !strings.Contains(result, "| 🔄 |") {
		t.Error("missing changes status icon")
	}
	if !strings.Contains(result, "| ❌ |") {
		t.Error("missing failed status icon")
	}

	if !strings.Contains(result, "<details>") {
		t.Error("missing expandable details")
	}

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

func TestFindTerraCIComment(t *testing.T) {
	notes := []*gitlab.Note{
		{ID: 1, Body: "Some other comment"},
		{ID: 2, Body: ci.CommentMarker + "\n\n## Terraform Plan"},
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
	notes := []*gitlab.Note{
		{ID: 1, Body: "Some comment"},
		{ID: 2, Body: "Another comment"},
	}

	found := FindTerraCIComment(notes)
	if found != nil {
		t.Error("expected nil, found a comment")
	}
}

func TestFindTerraCIComment_NilNotes(t *testing.T) {
	found := FindTerraCIComment(nil)
	if found != nil {
		t.Error("expected nil for nil notes input")
	}
}

func TestFindTerraCIComment_EmptyNotes(t *testing.T) {
	found := FindTerraCIComment([]*gitlab.Note{})
	if found != nil {
		t.Error("expected nil for empty notes input")
	}
}

func TestFindTerraCIComment_FirstMatch(t *testing.T) {
	notes := []*gitlab.Note{
		{ID: 1, Body: ci.CommentMarker + " first"},
		{ID: 2, Body: ci.CommentMarker + " second"},
	}

	found := FindTerraCIComment(notes)
	if found == nil {
		t.Fatal("expected to find comment")
	}
	if found.ID != 1 {
		t.Errorf("expected first match (ID=1), got ID=%d", found.ID)
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
			result := ci.Truncate(tt.input, tt.maxLen)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}
