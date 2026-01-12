package gitlab

import (
	"strings"
	"testing"
)

func TestCommentRenderer_RenderWithCost(t *testing.T) {
	renderer := NewCommentRenderer()

	plans := []ModulePlan{
		{
			ModuleID:    "platform/prod/eu-central-1/vpc",
			Environment: "prod",
			Status:      PlanStatusChanges,
			Summary:     "+2 ~1",
			CostBefore:  100.50,
			CostAfter:   150.75,
			CostDiff:    50.25,
			HasCost:     true,
		},
		{
			ModuleID:    "platform/prod/eu-central-1/eks",
			Environment: "prod",
			Status:      PlanStatusNoChanges,
			Summary:     "No changes",
			CostBefore:  73.00,
			CostAfter:   73.00,
			CostDiff:    0,
			HasCost:     true,
		},
	}

	data := &CommentData{
		Plans: plans,
	}

	result := renderer.Render(data)

	// Check that Cost column is present
	if !strings.Contains(result, "| Cost |") {
		t.Error("Expected Cost column header")
	}

	// Check cost formatting
	if !strings.Contains(result, "$100.50") {
		t.Error("Expected $100.50 in output")
	}
	if !strings.Contains(result, "+$50.25") {
		t.Error("Expected +$50.25 in output")
	}
	if !strings.Contains(result, "$150.75") {
		t.Error("Expected $150.75 in output")
	}

	t.Log("Rendered comment:\n" + result)
}

func TestCommentRenderer_RenderWithoutCost(t *testing.T) {
	renderer := NewCommentRenderer()

	plans := []ModulePlan{
		{
			ModuleID:    "platform/prod/eu-central-1/vpc",
			Environment: "prod",
			Status:      PlanStatusChanges,
			Summary:     "+2 ~1",
			HasCost:     false, // No cost data
		},
	}

	data := &CommentData{
		Plans: plans,
	}

	result := renderer.Render(data)

	// Check that Cost column is NOT present
	if strings.Contains(result, "| Cost |") {
		t.Error("Cost column should not be present when no cost data")
	}
}

func TestFormatCostCell(t *testing.T) {
	tests := []struct {
		name     string
		plan     ModulePlan
		expected string
	}{
		{
			name:     "no cost",
			plan:     ModulePlan{HasCost: false},
			expected: "-",
		},
		{
			name: "no change",
			plan: ModulePlan{
				HasCost:    true,
				CostBefore: 100,
				CostAfter:  100,
				CostDiff:   0,
			},
			expected: "$100.00",
		},
		{
			name: "increase",
			plan: ModulePlan{
				HasCost:    true,
				CostBefore: 100,
				CostAfter:  150,
				CostDiff:   50,
			},
			expected: "$100.00 +$50.00 → $150.00",
		},
		{
			name: "decrease",
			plan: ModulePlan{
				HasCost:    true,
				CostBefore: 150,
				CostAfter:  100,
				CostDiff:   -50,
			},
			expected: "$150.00 -$50.00 → $100.00",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatCostCell(&tt.plan)
			if result != tt.expected {
				t.Errorf("formatCostCell() = %q, want %q", result, tt.expected)
			}
		})
	}
}
