package summaryengine

import (
	"strings"
	"testing"
	"time"

	"github.com/edelwud/terraci/pkg/ci"
)

func TestComposeComment_BasicPlans(t *testing.T) {
	t.Parallel()

	plans := []ci.ModulePlan{
		{
			ModuleID:   "svc/prod/us-east-1/vpc",
			Components: map[string]string{"environment": "prod"},
			Status:     ci.PlanStatusChanges,
			Summary:    "+2 ~1 -0",
		},
		{
			ModuleID:   "svc/prod/us-east-1/rds",
			Components: map[string]string{"environment": "prod"},
			Status:     ci.PlanStatusNoChanges,
			Summary:    "No changes",
		},
		{
			ModuleID:   "svc/staging/us-east-1/vpc",
			Components: map[string]string{"environment": "staging"},
			Status:     ci.PlanStatusFailed,
			Error:      "init failed",
		},
	}

	result := ComposeComment(plans, nil, "", "", time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC))

	if !strings.Contains(result, ci.CommentMarker) {
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

func TestComposeComment_EmptyPlans(t *testing.T) {
	t.Parallel()

	result := ComposeComment([]ci.ModulePlan{}, nil, "", "", time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC))

	if !strings.Contains(result, "**0** modules analyzed") {
		t.Errorf("expected '**0** modules analyzed', got: %s", result)
	}
}

func TestComposeComment_WithReport(t *testing.T) {
	t.Parallel()

	plans := []ci.ModulePlan{
		{
			ModuleID:   "svc/prod/us-east-1/vpc",
			Components: map[string]string{"environment": "prod"},
			Status:     ci.PlanStatusChanges,
			Summary:    "+1",
		},
	}

	reports := []*ci.Report{
		{
			Plugin:  "policy",
			Title:   "Policy Check",
			Status:  ci.ReportStatusFail,
			Summary: "2 modules: 1 passed, 0 warned, 1 failed",
			Sections: []ci.ReportSection{{
				Kind:           ci.ReportSectionKindFindings,
				Title:          "Policy Check",
				Status:         ci.ReportStatusFail,
				SectionSummary: "2 modules: 1 passed, 0 warned, 1 failed",
				Findings: &ci.FindingsSection{
					Rows: []ci.FindingRow{{
						ModulePath: "svc/prod/us-east-1/vpc",
						Status:     ci.FindingRowStatusFail,
						Findings: []ci.Finding{{
							Severity:  ci.FindingSeverityFail,
							Message:   "too expensive",
							Namespace: "terraform.cost",
						}},
					}},
				},
			}},
		},
	}

	result := ComposeComment(plans, reports, "", "", time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC))

	if !strings.Contains(result, "Policy Check") {
		t.Error("expected 'Policy Check' section")
	}
	if !strings.Contains(result, "too expensive") {
		t.Error("expected failure message in output")
	}
	if !strings.Contains(result, "1 failed") {
		t.Error("expected failure count in summary")
	}
}

func TestComposeComment_WithCostData(t *testing.T) {
	t.Parallel()

	plans := []ci.ModulePlan{
		{
			ModuleID:   "svc/prod/us-east-1/vpc",
			Components: map[string]string{"environment": "prod"},
			Status:     ci.PlanStatusChanges,
			Summary:    "+1",
			HasCost:    true,
			CostBefore: 10.0,
			CostAfter:  15.0,
			CostDiff:   5.0,
		},
	}

	result := ComposeComment(plans, nil, "", "", time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC))

	if !strings.Contains(result, "| Cost |") {
		t.Error("expected 'Cost' column header in table")
	}
	if !strings.Contains(result, "$10.00") {
		t.Errorf("expected cost before in output, got: %s", result)
	}
}

func TestComposeComment_FiltersEnvironmentPlansToChangedAndFailed(t *testing.T) {
	t.Parallel()

	plans := []ci.ModulePlan{
		{
			ModuleID:   "svc/prod/us-east-1/vpc",
			Components: map[string]string{"environment": "prod"},
			Status:     ci.PlanStatusChanges,
			Summary:    "+1",
		},
		{
			ModuleID:   "svc/prod/us-east-1/rds",
			Components: map[string]string{"environment": "prod"},
			Status:     ci.PlanStatusNoChanges,
			Summary:    "No changes",
		},
		{
			ModuleID:   "svc/prod/us-east-1/iam",
			Components: map[string]string{"environment": "prod"},
			Status:     ci.PlanStatusFailed,
			Error:      "apply failed",
		},
	}

	result := ComposeComment(plans, nil, "", "", time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC))

	if !strings.Contains(result, "svc/prod/us-east-1/vpc") {
		t.Fatalf("expected changed module in output: %s", result)
	}
	if !strings.Contains(result, "svc/prod/us-east-1/iam") {
		t.Fatalf("expected failed module in output: %s", result)
	}
	if strings.Contains(result, "svc/prod/us-east-1/rds") {
		t.Fatalf("unexpected unchanged module in output: %s", result)
	}
}

func TestComposeComment_FiltersCostReportToAddedCosts(t *testing.T) {
	t.Parallel()

	reports := []*ci.Report{{
		Plugin:  "cost",
		Title:   "Cost Estimation",
		Status:  ci.ReportStatusWarn,
		Summary: "3 modules, total: $27.00/mo (diff: +5.00)",
		Sections: []ci.ReportSection{{
			Kind:           ci.ReportSectionKindCostChanges,
			Title:          "Cost Estimation",
			Status:         ci.ReportStatusWarn,
			SectionSummary: "3 modules, total: $27.00/mo (diff: +5.00)",
			CostChanges: &ci.CostChangesSection{
				Totals: ci.CostTotals{After: 37, Diff: -5},
				Rows: []ci.CostChangeRow{
					{ModulePath: "svc/prod/us-east-1/vpc", Before: 10, After: 15, Diff: 5, HasCost: true},
					{ModulePath: "svc/prod/us-east-1/rds", Before: 12, After: 12, Diff: 0, HasCost: true},
					{ModulePath: "svc/prod/us-east-1/redis", Before: 20, After: 10, Diff: -10, HasCost: true},
				},
			},
		}},
	}}

	result := ComposeComment(nil, reports, "", "", time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC))

	if !strings.Contains(result, "svc/prod/us-east-1/vpc") {
		t.Fatalf("expected positive diff row in output: %s", result)
	}
	if strings.Contains(result, "svc/prod/us-east-1/rds") {
		t.Fatalf("unexpected zero diff row in output: %s", result)
	}
	if strings.Contains(result, "svc/prod/us-east-1/redis") {
		t.Fatalf("unexpected negative diff row in output: %s", result)
	}
}

func TestComposeComment_FiltersTfupdateReportToUpdatableModules(t *testing.T) {
	t.Parallel()

	reports := []*ci.Report{{
		Plugin:  "tfupdate",
		Title:   "Dependency Update Check",
		Status:  ci.ReportStatusWarn,
		Summary: "4 checked, 2 updates available, 0 applied, 0 errors",
		Sections: []ci.ReportSection{{
			Kind:           ci.ReportSectionKindDependencyUpdates,
			Title:          "Dependency Update Check",
			Status:         ci.ReportStatusWarn,
			SectionSummary: "4 checked, 2 updates available, 0 applied, 0 errors",
			DependencyUpdates: &ci.DependencyUpdatesSection{
				Rows: []ci.DependencyUpdateRow{
					{ModulePath: "svc/prod/us-east-1/vpc", Kind: ci.DependencyKindProvider, Name: "hashicorp/aws", Current: "~> 5.0", Latest: "5.4.0", Status: ci.DependencyUpdateStatusUpdateAvailable},
					{ModulePath: "svc/prod/us-east-1/rds", Kind: ci.DependencyKindProvider, Name: "hashicorp/random", Current: "3.0.0", Latest: "3.0.0", Status: ci.DependencyUpdateStatusUpToDate},
					{ModulePath: "svc/prod/us-east-1/eks", Kind: ci.DependencyKindModule, Name: "terraform-aws-modules/eks/aws", Current: "20.0.0", Latest: "21.0.0", Status: ci.DependencyUpdateStatusApplied},
					{ModulePath: "svc/prod/us-east-1/iam", Kind: ci.DependencyKindModule, Name: "terraform-aws-modules/iam/aws", Current: "1.0.0", Latest: "1.0.0", Status: ci.DependencyUpdateStatusUpToDate},
				},
			},
		}},
	}}

	result := ComposeComment(nil, reports, "", "", time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC))

	if !strings.Contains(result, "svc/prod/us-east-1/vpc") {
		t.Fatalf("expected update available provider row in output: %s", result)
	}
	if !strings.Contains(result, "svc/prod/us-east-1/eks") {
		t.Fatalf("expected applied module row in output: %s", result)
	}
	if strings.Contains(result, "svc/prod/us-east-1/rds") {
		t.Fatalf("unexpected up-to-date provider row in output: %s", result)
	}
	if strings.Contains(result, "svc/prod/us-east-1/iam") {
		t.Fatalf("unexpected up-to-date module row in output: %s", result)
	}
}

func TestComposeComment_CommitSHA(t *testing.T) {
	t.Parallel()

	result := ComposeComment([]ci.ModulePlan{}, nil, "abcdef1234567890abcdef1234567890abcdef12", "", time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC))

	if !strings.Contains(result, "`abcdef12`") {
		t.Errorf("expected truncated SHA 'abcdef12', got: %s", result)
	}
	if strings.Contains(result, "abcdef1234567890") {
		t.Error("full SHA should not appear in output")
	}
}

func TestCalculateStats(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		plans []ci.ModulePlan
		want  planStats
	}{
		{
			name:  "empty",
			plans: nil,
			want:  planStats{},
		},
		{
			name: "all statuses",
			plans: []ci.ModulePlan{
				{Status: ci.PlanStatusSuccess},
				{Status: ci.PlanStatusNoChanges},
				{Status: ci.PlanStatusChanges},
				{Status: ci.PlanStatusFailed},
				{Status: ci.PlanStatusPending},
				{Status: ci.PlanStatusRunning},
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
			plans: []ci.ModulePlan{
				{Status: ci.PlanStatusNoChanges},
				{Status: ci.PlanStatusNoChanges},
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
			t.Parallel()

			got := calculateStats(tt.plans)
			if got != tt.want {
				t.Errorf("calculateStats() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestRenderStats(t *testing.T) {
	t.Parallel()

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
			t.Parallel()

			got := renderStats(tt.stats)
			for _, want := range tt.contains {
				if !strings.Contains(got, want) {
					t.Errorf("renderStats() = %q, want to contain %q", got, want)
				}
			}
		})
	}
}

func TestGroupByEnvironment(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		plans    []ci.ModulePlan
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
			plans: []ci.ModulePlan{
				{Components: map[string]string{"environment": "prod"}},
				{Components: map[string]string{"environment": "prod"}},
				{Components: map[string]string{"environment": "staging"}},
			},
			wantKeys: []string{"prod", "staging"},
			wantLen:  map[string]int{"prod": 2, "staging": 1},
		},
		{
			name: "empty env becomes default",
			plans: []ci.ModulePlan{
				{Components: map[string]string{}},
				{Components: map[string]string{"environment": "prod"}},
			},
			wantKeys: []string{"default", "prod"},
			wantLen:  map[string]int{"default": 1, "prod": 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := groupByEnvironment(tt.plans)
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
	t.Parallel()

	tests := []struct {
		name string
		plan ci.ModulePlan
		want string
	}{
		{
			name: "no cost",
			plan: ci.ModulePlan{HasCost: false},
			want: "-",
		},
		{
			name: "zero diff",
			plan: ci.ModulePlan{HasCost: true, CostAfter: 25.0, CostDiff: 0},
			want: "$25.00",
		},
		{
			name: "positive diff",
			plan: ci.ModulePlan{HasCost: true, CostBefore: 10.0, CostAfter: 15.0, CostDiff: 5.0},
			want: "$10.00 +$5.00 -> $15.00",
		},
		{
			name: "negative diff",
			plan: ci.ModulePlan{HasCost: true, CostBefore: 20.0, CostAfter: 10.0, CostDiff: -10.0},
			want: "$20.00 -$10.00 -> $10.00",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := FormatCostCell(&tt.plan)
			if got != tt.want {
				t.Errorf("FormatCostCell() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatMonthlyCost(t *testing.T) {
	t.Parallel()

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
			t.Parallel()

			got := FormatMonthlyCost(tt.cost)
			if got != tt.want {
				t.Errorf("FormatMonthlyCost(%v) = %q, want %q", tt.cost, got, tt.want)
			}
		})
	}
}

func TestFormatCostDiff(t *testing.T) {
	t.Parallel()

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
			t.Parallel()

			got := FormatCostDiff(tt.diff)
			if got != tt.want {
				t.Errorf("FormatCostDiff(%v) = %q, want %q", tt.diff, got, tt.want)
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	t.Parallel()

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
			t.Parallel()

			got := Truncate(tt.s, tt.maxLen)
			if got != tt.want {
				t.Errorf("Truncate(%q, %d) = %q, want %q", tt.s, tt.maxLen, got, tt.want)
			}
		})
	}
}

func TestStatusIcon(t *testing.T) {
	t.Parallel()

	tests := []struct {
		status ci.PlanStatus
		want   string
	}{
		{ci.PlanStatusSuccess, "ok"},
		{ci.PlanStatusNoChanges, "ok"},
		{ci.PlanStatusChanges, "changes"},
		{ci.PlanStatusFailed, "failed"},
		{ci.PlanStatusPending, "pending"},
		{ci.PlanStatusRunning, "running"},
		{ci.PlanStatus("unknown"), "?"},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			t.Parallel()

			got := statusIcon(tt.status)
			if got != tt.want {
				t.Errorf("statusIcon(%q) = %q, want %q", tt.status, got, tt.want)
			}
		})
	}
}

func TestRenderExpandableDetails(t *testing.T) {
	t.Parallel()

	t.Run("with structured details", func(t *testing.T) {
		t.Parallel()

		p := &ci.ModulePlan{
			ModuleID:          "svc/prod/us-east-1/vpc",
			Status:            ci.PlanStatusChanges,
			Summary:           "+2 ~1 -0",
			StructuredDetails: "### Resources\n- aws_vpc.main (create)",
		}

		got := renderExpandableDetails(p)

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
		t.Parallel()

		p := &ci.ModulePlan{
			ModuleID:      "svc/prod/us-east-1/rds",
			Status:        ci.PlanStatusChanges,
			Summary:       "No changes",
			RawPlanOutput: "+ resource \"aws_db_instance\" \"main\"",
		}

		got := renderExpandableDetails(p)

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
		t.Parallel()

		p := &ci.ModulePlan{
			ModuleID: "svc/prod/us-east-1/eks",
			Status:   ci.PlanStatusFailed,
			Error:    "terraform init failed",
		}

		got := renderExpandableDetails(p)

		if !strings.Contains(got, "FAILED svc/prod/us-east-1/eks") {
			t.Errorf("expected failed prefix with module ID, got: %s", got)
		}
	})

	t.Run("no summary uses plain title", func(t *testing.T) {
		t.Parallel()

		p := &ci.ModulePlan{
			ModuleID:          "svc/prod/us-east-1/vpc",
			Status:            ci.PlanStatusChanges,
			Summary:           "",
			StructuredDetails: "some details",
		}

		got := renderExpandableDetails(p)

		// With empty summary, title should just be the module ID
		if !strings.Contains(got, "svc/prod/us-east-1/vpc</summary>") {
			t.Errorf("expected plain title without summary parens, got: %s", got)
		}
	})
}

func TestRenderReportSection(t *testing.T) {
	t.Parallel()

	t.Run("with fail status", func(t *testing.T) {
		t.Parallel()

		report := &ci.Report{
			Plugin:  "policy",
			Title:   "Policy Check",
			Status:  ci.ReportStatusFail,
			Summary: "3 modules: 1 passed, 0 warned, 2 failed",
			Sections: []ci.ReportSection{{
				Kind:           ci.ReportSectionKindFindings,
				Title:          "Policy Check",
				Status:         ci.ReportStatusFail,
				SectionSummary: "3 modules: 1 passed, 0 warned, 2 failed",
				Findings: &ci.FindingsSection{
					Rows: []ci.FindingRow{{
						ModulePath: "svc/prod/us-east-1/vpc",
						Status:     ci.FindingRowStatusFail,
						Findings: []ci.Finding{{
							Severity:  ci.FindingSeverityFail,
							Message:   "no public access",
							Namespace: "terraform.security",
						}},
					}},
				},
			}},
		}

		got := renderReportSection(report)

		if !strings.Contains(got, "Policy Check") {
			t.Error("expected policy check header")
		}
		if !strings.Contains(got, "no public access") {
			t.Error("expected failure message")
		}
		if !strings.Contains(got, "2 failed") {
			t.Error("expected failure count")
		}
	})

	t.Run("with warn status", func(t *testing.T) {
		t.Parallel()

		report := &ci.Report{
			Plugin:  "policy",
			Title:   "Policy Check",
			Status:  ci.ReportStatusWarn,
			Summary: "1 modules: 0 passed, 1 warned, 0 failed",
			Sections: []ci.ReportSection{{
				Kind:           ci.ReportSectionKindFindings,
				Title:          "Policy Check",
				Status:         ci.ReportStatusWarn,
				SectionSummary: "1 modules: 0 passed, 1 warned, 0 failed",
				Findings: &ci.FindingsSection{
					Rows: []ci.FindingRow{{
						ModulePath: "svc/staging/us-east-1/vpc",
						Status:     ci.FindingRowStatusWarn,
						Findings: []ci.Finding{{
							Severity:  ci.FindingSeverityWarn,
							Message:   "non-standard naming",
							Namespace: "terraform.naming",
						}},
					}},
				},
			}},
		}

		got := renderReportSection(report)

		if !strings.Contains(got, "warn") {
			t.Error("expected warning status")
		}
		if !strings.Contains(got, "non-standard naming") {
			t.Error("expected warning message")
		}
	})

	t.Run("pass status", func(t *testing.T) {
		t.Parallel()

		report := &ci.Report{
			Plugin:  "policy",
			Title:   "Policy Check",
			Status:  ci.ReportStatusPass,
			Summary: "2 modules: 2 passed, 0 warned, 0 failed",
		}

		got := renderReportSection(report)

		if !strings.Contains(got, "pass") {
			t.Error("expected pass status")
		}
	})
}

func TestRenderPlanRow(t *testing.T) {
	t.Parallel()

	t.Run("without cost", func(t *testing.T) {
		t.Parallel()

		p := &ci.ModulePlan{
			ModuleID: "svc/prod/us-east-1/vpc",
			Status:   ci.PlanStatusChanges,
			Summary:  "+2 ~1 -0",
		}

		got := renderPlanRow(p, false)

		if !strings.Contains(got, "changes") {
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
		t.Parallel()

		p := &ci.ModulePlan{
			ModuleID:  "svc/prod/us-east-1/vpc",
			Status:    ci.PlanStatusNoChanges,
			Summary:   "No changes",
			HasCost:   true,
			CostAfter: 50.0,
			CostDiff:  0,
		}

		got := renderPlanRow(p, true)

		// Should have 4 columns
		if strings.Count(got, "|") != 5 {
			t.Errorf("expected 5 pipe chars for 4-column row, got %d in %q", strings.Count(got, "|"), got)
		}
		if !strings.Contains(got, "$50.00") {
			t.Error("expected cost value")
		}
	})

	t.Run("with error", func(t *testing.T) {
		t.Parallel()

		p := &ci.ModulePlan{
			ModuleID: "svc/prod/us-east-1/vpc",
			Status:   ci.PlanStatusFailed,
			Error:    "init failed: something went wrong",
		}

		got := renderPlanRow(p, false)

		if !strings.Contains(got, "init failed: something went wrong") {
			t.Error("expected error message in summary")
		}
	})

	t.Run("empty summary becomes dash", func(t *testing.T) {
		t.Parallel()

		p := &ci.ModulePlan{
			ModuleID: "svc/prod/us-east-1/vpc",
			Status:   ci.PlanStatusPending,
		}

		got := renderPlanRow(p, false)

		if !strings.Contains(got, "| - |") {
			t.Errorf("expected dash for empty summary, got: %s", got)
		}
	})
}
