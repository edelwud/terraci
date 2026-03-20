package ci

import (
	"strings"
	"testing"
	"time"

	"github.com/edelwud/terraci/internal/policy"
)

func TestRender_BasicPlans(t *testing.T) {
	r := NewCommentRenderer()
	data := &CommentData{
		Plans: []ModulePlan{
			{
				ModuleID:   "svc/prod/us-east-1/vpc",
				Components: map[string]string{"environment": "prod"},
				Status:     PlanStatusChanges,
				Summary:    "+2 ~1 -0",
			},
			{
				ModuleID:   "svc/prod/us-east-1/rds",
				Components: map[string]string{"environment": "prod"},
				Status:     PlanStatusNoChanges,
				Summary:    "No changes",
			},
			{
				ModuleID:   "svc/staging/us-east-1/vpc",
				Components: map[string]string{"environment": "staging"},
				Status:     PlanStatusFailed,
				Error:      "init failed",
			},
		},
		GeneratedAt: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC),
	}

	result := r.Render(data)

	if !strings.Contains(result, CommentMarker) {
		t.Error("expected CommentMarker in output")
	}
	if !strings.Contains(result, "Terraform Plan Summary") {
		t.Error("expected 'Terraform Plan Summary' header")
	}
	if !strings.Contains(result, "`prod`") {
		t.Error("expected prod environment section")
	}
	if !strings.Contains(result, "`staging`") {
		t.Error("expected staging environment section")
	}
	// Stats line: 3 modules with mixed statuses
	if !strings.Contains(result, "**3** modules:") {
		t.Error("expected stats line with 3 modules")
	}
}

func TestRender_EmptyPlans(t *testing.T) {
	r := NewCommentRenderer()
	data := &CommentData{
		Plans:       []ModulePlan{},
		GeneratedAt: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC),
	}

	result := r.Render(data)

	if !strings.Contains(result, "**0** modules analyzed") {
		t.Errorf("expected '**0** modules analyzed', got: %s", result)
	}
}

func TestRender_WithPolicySummary(t *testing.T) {
	r := NewCommentRenderer()
	data := &CommentData{
		Plans: []ModulePlan{
			{
				ModuleID:   "svc/prod/us-east-1/vpc",
				Components: map[string]string{"environment": "prod"},
				Status:     PlanStatusChanges,
				Summary:    "+1",
			},
		},
		PolicySummary: &policy.Summary{
			TotalModules:  2,
			PassedModules: 1,
			FailedModules: 1,
			TotalFailures: 1,
			Results: []policy.Result{
				{
					Module: "svc/prod/us-east-1/vpc",
					Failures: []policy.Violation{
						{Namespace: "terraform.cost", Message: "too expensive"},
					},
				},
			},
		},
		GeneratedAt: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC),
	}

	result := r.Render(data)

	if !strings.Contains(result, "Policy Check") {
		t.Error("expected 'Policy Check' section")
	}
	if !strings.Contains(result, "too expensive") {
		t.Error("expected failure message in output")
	}
	if !strings.Contains(result, "Failures (1)") {
		t.Error("expected failure count")
	}
}

func TestRender_WithCostData(t *testing.T) {
	r := NewCommentRenderer()
	data := &CommentData{
		Plans: []ModulePlan{
			{
				ModuleID:   "svc/prod/us-east-1/vpc",
				Components: map[string]string{"environment": "prod"},
				Status:     PlanStatusChanges,
				Summary:    "+1",
				HasCost:    true,
				CostBefore: 10.0,
				CostAfter:  15.0,
				CostDiff:   5.0,
			},
		},
		GeneratedAt: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC),
	}

	result := r.Render(data)

	if !strings.Contains(result, "| Cost |") {
		t.Error("expected 'Cost' column header in table")
	}
	if !strings.Contains(result, "$10.00") {
		t.Errorf("expected cost before in output, got: %s", result)
	}
}

func TestRender_CommitSHA(t *testing.T) {
	r := NewCommentRenderer()
	data := &CommentData{
		Plans:       []ModulePlan{},
		CommitSHA:   "abcdef1234567890abcdef1234567890abcdef12",
		GeneratedAt: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC),
	}

	result := r.Render(data)

	if !strings.Contains(result, "`abcdef12`") {
		t.Errorf("expected truncated SHA 'abcdef12', got: %s", result)
	}
	if strings.Contains(result, "abcdef1234567890") {
		t.Error("full SHA should not appear in output")
	}
}

func TestCalculateStats(t *testing.T) {
	r := NewCommentRenderer()

	tests := []struct {
		name  string
		plans []ModulePlan
		want  planStats
	}{
		{
			name:  "empty",
			plans: nil,
			want:  planStats{},
		},
		{
			name: "all statuses",
			plans: []ModulePlan{
				{Status: PlanStatusSuccess},
				{Status: PlanStatusNoChanges},
				{Status: PlanStatusChanges},
				{Status: PlanStatusFailed},
				{Status: PlanStatusPending},
				{Status: PlanStatusRunning},
			},
			want: planStats{
				Total:     6,
				Success:   3, // NoChanges(2) + Changes(1)
				NoChanges: 2, // success + no_changes
				Changes:   1,
				Failed:    1,
				Pending:   1,
				Running:   1,
			},
		},
		{
			name: "all no changes",
			plans: []ModulePlan{
				{Status: PlanStatusNoChanges},
				{Status: PlanStatusNoChanges},
			},
			want: planStats{
				Total:     2,
				Success:   2,
				NoChanges: 2,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := r.calculateStats(tt.plans)
			if got != tt.want {
				t.Errorf("calculateStats() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestRenderStats(t *testing.T) {
	r := NewCommentRenderer()

	tests := []struct {
		name     string
		stats    planStats
		contains []string
	}{
		{
			name:     "zero total",
			stats:    planStats{Total: 0},
			contains: []string{"**0** modules analyzed"},
		},
		{
			name:     "only no changes",
			stats:    planStats{Total: 3, NoChanges: 3},
			contains: []string{"**3** modules:", "**3** no changes"},
		},
		{
			name:     "mixed",
			stats:    planStats{Total: 5, Changes: 2, NoChanges: 1, Failed: 1, Pending: 1},
			contains: []string{"**5** modules:", "**2** with changes", "**1** no changes", "**1** failed", "**1** pending"},
		},
		{
			name:     "with running",
			stats:    planStats{Total: 1, Running: 1},
			contains: []string{"**1** modules:", "**1** running"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := r.renderStats(tt.stats)
			for _, want := range tt.contains {
				if !strings.Contains(got, want) {
					t.Errorf("renderStats() = %q, want to contain %q", got, want)
				}
			}
		})
	}
}

func TestGroupByEnvironment(t *testing.T) {
	r := NewCommentRenderer()

	tests := []struct {
		name     string
		plans    []ModulePlan
		wantKeys []string
		wantLen  map[string]int
	}{
		{
			name:     "empty",
			plans:    nil,
			wantKeys: nil,
			wantLen:  map[string]int{},
		},
		{
			name: "multiple envs",
			plans: []ModulePlan{
				{Components: map[string]string{"environment": "prod"}},
				{Components: map[string]string{"environment": "prod"}},
				{Components: map[string]string{"environment": "staging"}},
			},
			wantKeys: []string{"prod", "staging"},
			wantLen:  map[string]int{"prod": 2, "staging": 1},
		},
		{
			name: "empty env becomes default",
			plans: []ModulePlan{
				{Components: map[string]string{}},
				{Components: map[string]string{"environment": "prod"}},
			},
			wantKeys: []string{"default", "prod"},
			wantLen:  map[string]int{"default": 1, "prod": 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := r.groupByEnvironment(tt.plans)
			for _, key := range tt.wantKeys {
				if _, ok := got[key]; !ok {
					t.Errorf("groupByEnvironment() missing key %q", key)
				}
			}
			for key, wantCount := range tt.wantLen {
				if len(got[key]) != wantCount {
					t.Errorf("groupByEnvironment()[%q] len = %d, want %d", key, len(got[key]), wantCount)
				}
			}
		})
	}
}

func TestFormatCostCell(t *testing.T) {
	tests := []struct {
		name string
		plan ModulePlan
		want string
	}{
		{
			name: "no cost",
			plan: ModulePlan{HasCost: false},
			want: "-",
		},
		{
			name: "zero diff",
			plan: ModulePlan{HasCost: true, CostAfter: 25.0, CostDiff: 0},
			want: "$25.00",
		},
		{
			name: "positive diff",
			plan: ModulePlan{HasCost: true, CostBefore: 10.0, CostAfter: 15.0, CostDiff: 5.0},
			want: "$10.00 +$5.00 → $15.00",
		},
		{
			name: "negative diff",
			plan: ModulePlan{HasCost: true, CostBefore: 20.0, CostAfter: 10.0, CostDiff: -10.0},
			want: "$20.00 -$10.00 → $10.00",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatCostCell(&tt.plan)
			if got != tt.want {
				t.Errorf("FormatCostCell() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatMonthlyCost(t *testing.T) {
	tests := []struct {
		name string
		cost float64
		want string
	}{
		{"zero", 0, "$0"},
		{"below threshold", 0.005, "<$0.01"},
		{"sub-dollar", 0.5, "$0.5000"},
		{"normal", 10.5, "$10.50"},
		{"thousand plus", 1500, "$1500"},
		{"exactly one", 1.0, "$1.00"},
		{"just under thousand", 999.99, "$999.99"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatMonthlyCost(tt.cost)
			if got != tt.want {
				t.Errorf("FormatMonthlyCost(%v) = %q, want %q", tt.cost, got, tt.want)
			}
		})
	}
}

func TestFormatCostDiff(t *testing.T) {
	tests := []struct {
		name string
		diff float64
		want string
	}{
		{"zero", 0, "$0"},
		{"positive normal", 5.0, "+$5.00"},
		{"negative normal", -5.0, "-$5.00"},
		{"positive sub-dollar", 0.5, "+$0.5000"},
		{"negative sub-dollar", -0.5, "-$0.5000"},
		{"positive thousand", 1500, "+$1500"},
		{"negative thousand", -1500, "-$1500"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatCostDiff(tt.diff)
			if got != tt.want {
				t.Errorf("FormatCostDiff(%v) = %q, want %q", tt.diff, got, tt.want)
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name   string
		s      string
		maxLen int
		want   string
	}{
		{
			name:   "short string unchanged",
			s:      "hello",
			maxLen: 10,
			want:   "hello",
		},
		{
			name:   "exact length unchanged",
			s:      "hello",
			maxLen: 5,
			want:   "hello",
		},
		{
			name:   "long string truncated",
			s:      "hello world, this is a long string",
			maxLen: 10,
			want:   "hello w...",
		},
		{
			name:   "empty string",
			s:      "",
			maxLen: 5,
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Truncate(tt.s, tt.maxLen)
			if got != tt.want {
				t.Errorf("Truncate(%q, %d) = %q, want %q", tt.s, tt.maxLen, got, tt.want)
			}
		})
	}
}

func TestStatusIcon(t *testing.T) {
	r := NewCommentRenderer()

	tests := []struct {
		status PlanStatus
		want   string
	}{
		{PlanStatusSuccess, "✅"},
		{PlanStatusNoChanges, "✅"},
		{PlanStatusChanges, "🔄"},
		{PlanStatusFailed, "❌"},
		{PlanStatusPending, "⏳"},
		{PlanStatusRunning, "🔄"},
		{PlanStatus("unknown"), "❓"},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			got := r.statusIcon(tt.status)
			if got != tt.want {
				t.Errorf("statusIcon(%q) = %q, want %q", tt.status, got, tt.want)
			}
		})
	}
}

func TestRenderExpandableDetails(t *testing.T) {
	r := NewCommentRenderer()

	t.Run("with structured details", func(t *testing.T) {
		p := &ModulePlan{
			ModuleID:          "svc/prod/us-east-1/vpc",
			Status:            PlanStatusChanges,
			Summary:           "+2 ~1 -0",
			StructuredDetails: "### Resources\n- aws_vpc.main (create)",
		}

		got := r.renderExpandableDetails(p)

		if !strings.Contains(got, "<details>") {
			t.Error("expected <details> tag")
		}
		if !strings.Contains(got, "svc/prod/us-east-1/vpc (+2 ~1 -0)") {
			t.Errorf("expected module ID with summary in title, got: %s", got)
		}
		if !strings.Contains(got, "### Resources") {
			t.Error("expected structured details content")
		}
	})

	t.Run("with raw plan output", func(t *testing.T) {
		p := &ModulePlan{
			ModuleID:      "svc/prod/us-east-1/rds",
			Status:        PlanStatusChanges,
			Summary:       "No changes",
			RawPlanOutput: "+ resource \"aws_db_instance\" \"main\"",
		}

		got := r.renderExpandableDetails(p)

		if !strings.Contains(got, "Full plan output") {
			t.Error("expected 'Full plan output' section")
		}
		if !strings.Contains(got, "```diff") {
			t.Error("expected diff code block")
		}
		// Summary is "No changes" which equals the special case check
		if !strings.Contains(got, "svc/prod/us-east-1/rds") {
			t.Error("expected module ID in title")
		}
	})

	t.Run("failed status", func(t *testing.T) {
		p := &ModulePlan{
			ModuleID: "svc/prod/us-east-1/eks",
			Status:   PlanStatusFailed,
			Error:    "terraform init failed",
		}

		got := r.renderExpandableDetails(p)

		if !strings.Contains(got, "❌ svc/prod/us-east-1/eks") {
			t.Errorf("expected failed icon with module ID, got: %s", got)
		}
	})

	t.Run("no summary uses plain title", func(t *testing.T) {
		p := &ModulePlan{
			ModuleID:          "svc/prod/us-east-1/vpc",
			Status:            PlanStatusChanges,
			Summary:           "",
			StructuredDetails: "some details",
		}

		got := r.renderExpandableDetails(p)

		// With empty summary, title should just be the module ID prefixed with icon
		if !strings.Contains(got, "📋 svc/prod/us-east-1/vpc</summary>") {
			t.Errorf("expected plain title without summary parens, got: %s", got)
		}
	})
}

func TestRenderPolicySection(t *testing.T) {
	r := NewCommentRenderer()

	t.Run("with failures", func(t *testing.T) {
		summary := &policy.Summary{
			TotalModules:  3,
			PassedModules: 1,
			FailedModules: 2,
			TotalFailures: 3,
			Results: []policy.Result{
				{
					Module: "svc/prod/us-east-1/vpc",
					Failures: []policy.Violation{
						{Namespace: "terraform.security", Message: "no public access"},
						{Namespace: "terraform.cost", Message: "budget exceeded"},
					},
				},
				{
					Module: "svc/prod/us-east-1/rds",
					Failures: []policy.Violation{
						{Namespace: "terraform.security", Message: "encryption required"},
					},
				},
			},
		}

		got := r.renderPolicySection(summary)

		if !strings.Contains(got, "❌ Policy Check") {
			t.Error("expected failure icon in policy check header")
		}
		if !strings.Contains(got, "**3** modules checked") {
			t.Error("expected total modules count")
		}
		if !strings.Contains(got, "no public access") {
			t.Error("expected failure message")
		}
		if !strings.Contains(got, "Failures (3)") {
			t.Error("expected failures count")
		}
	})

	t.Run("with warnings only", func(t *testing.T) {
		summary := &policy.Summary{
			TotalModules:  1,
			WarnedModules: 1,
			TotalWarnings: 1,
			Results: []policy.Result{
				{
					Module: "svc/staging/us-east-1/vpc",
					Warnings: []policy.Violation{
						{Namespace: "terraform.naming", Message: "non-standard naming"},
					},
				},
			},
		}

		got := r.renderPolicySection(summary)

		if !strings.Contains(got, "⚠️ Policy Check") {
			t.Error("expected warning icon in policy check header")
		}
		if !strings.Contains(got, "Warnings (1)") {
			t.Error("expected warnings count")
		}
		if !strings.Contains(got, "non-standard naming") {
			t.Error("expected warning message")
		}
	})

	t.Run("all passed", func(t *testing.T) {
		summary := &policy.Summary{
			TotalModules:  2,
			PassedModules: 2,
		}

		got := r.renderPolicySection(summary)

		if !strings.Contains(got, "✅ Policy Check") {
			t.Error("expected pass icon in policy check header")
		}
	})
}

func TestRenderPlanRow(t *testing.T) {
	r := NewCommentRenderer()

	t.Run("without cost", func(t *testing.T) {
		p := &ModulePlan{
			ModuleID: "svc/prod/us-east-1/vpc",
			Status:   PlanStatusChanges,
			Summary:  "+2 ~1 -0",
		}

		got := r.renderPlanRow(p, false)

		if !strings.Contains(got, "🔄") {
			t.Error("expected changes icon")
		}
		if !strings.Contains(got, "`svc/prod/us-east-1/vpc`") {
			t.Error("expected module ID in backticks")
		}
		if !strings.Contains(got, "+2 ~1 -0") {
			t.Error("expected summary")
		}
		// Should have 3 columns (no cost)
		if strings.Count(got, "|") != 4 { // |status|module|summary|
			t.Errorf("expected 4 pipe chars for 3-column row, got %d in %q", strings.Count(got, "|"), got)
		}
	})

	t.Run("with cost", func(t *testing.T) {
		p := &ModulePlan{
			ModuleID:  "svc/prod/us-east-1/vpc",
			Status:    PlanStatusNoChanges,
			Summary:   "No changes",
			HasCost:   true,
			CostAfter: 50.0,
			CostDiff:  0,
		}

		got := r.renderPlanRow(p, true)

		// Should have 4 columns
		if strings.Count(got, "|") != 5 {
			t.Errorf("expected 5 pipe chars for 4-column row, got %d in %q", strings.Count(got, "|"), got)
		}
		if !strings.Contains(got, "$50.00") {
			t.Error("expected cost value")
		}
	})

	t.Run("with error", func(t *testing.T) {
		p := &ModulePlan{
			ModuleID: "svc/prod/us-east-1/vpc",
			Status:   PlanStatusFailed,
			Error:    "init failed: something went wrong",
		}

		got := r.renderPlanRow(p, false)

		if !strings.Contains(got, "init failed: something went wrong") {
			t.Error("expected error message in summary")
		}
	})

	t.Run("empty summary becomes dash", func(t *testing.T) {
		p := &ModulePlan{
			ModuleID: "svc/prod/us-east-1/vpc",
			Status:   PlanStatusPending,
		}

		got := r.renderPlanRow(p, false)

		if !strings.Contains(got, "| - |") {
			t.Errorf("expected dash for empty summary, got: %s", got)
		}
	})
}
