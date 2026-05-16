package summaryengine

import (
	"strings"
	"testing"
	"time"

	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/ci/citest"
)

func mustComposeComment(t *testing.T, plans []ci.PlanResult, reports []*ci.Report, commitSHA string, generatedAt time.Time) string {
	t.Helper()
	result, err := ComposeComment(plans, reports, commitSHA, "", generatedAt)
	if err != nil {
		t.Fatalf("ComposeComment() error = %v", err)
	}
	return result
}

func mustComposeCommentWithOptions(t *testing.T, plans []ci.PlanResult, reports []*ci.Report, commitSHA, pipelineID string, generatedAt time.Time, includeDetails bool) string {
	t.Helper()
	result, err := ComposeCommentWithOptions(plans, reports, commitSHA, pipelineID, generatedAt, includeDetails)
	if err != nil {
		t.Fatalf("ComposeCommentWithOptions() error = %v", err)
	}
	return result
}

func TestComposeComment_BasicPlans(t *testing.T) {
	t.Parallel()

	plans := []ci.PlanResult{
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

	result := mustComposeComment(t, plans, nil, "", time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC))

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

	result := mustComposeComment(t, []ci.PlanResult{}, nil, "", time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC))

	if !strings.Contains(result, "**0** modules analyzed") {
		t.Errorf("expected '**0** modules analyzed', got: %s", result)
	}
}

func TestComposeComment_WithReport(t *testing.T) {
	t.Parallel()

	plans := []ci.PlanResult{
		{
			ModuleID:   "svc/prod/us-east-1/vpc",
			Components: map[string]string{"environment": "prod"},
			Status:     ci.PlanStatusChanges,
			Summary:    "+1",
		},
	}

	reports := []*ci.Report{
		{
			Producer: "policy",
			Title:    "Policy Check",
			Status:   ci.ReportStatusFail,
			Summary:  "2 modules: 1 passed, 0 warned, 1 failed",
			Sections: []ci.ReportSection{citest.MustRenderedSection(
				"Policy Check",
				"2 modules: 1 passed, 0 warned, 1 failed",
				ci.ReportStatusFail,
				ci.RenderTableBlock("", []string{"Module", "Severity", "Namespace", "Message"}, [][]string{{
					"svc/prod/us-east-1/vpc",
					"fail",
					"terraform.cost",
					"too expensive",
				}}),
			)},
		},
	}

	result := mustComposeComment(t, plans, reports, "", time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC))

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

func TestComposeCommentWithOptions_WithoutDetailsOmitsPlanBody(t *testing.T) {
	t.Parallel()

	plans := []ci.PlanResult{{
		ModuleID:          "svc/prod/us-east-1/vpc",
		Components:        map[string]string{"environment": "prod"},
		Status:            ci.PlanStatusChanges,
		Summary:           "+1",
		StructuredDetails: "### Resources\n- aws_vpc.main (create)",
		RawPlanOutput:     "+ resource \"aws_vpc\" \"main\"",
	}}

	result := mustComposeCommentWithOptions(t, plans, nil, "", "", time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC), false)

	if strings.Contains(result, "### Resources") {
		t.Fatalf("comment should omit structured details when includeDetails=false:\n%s", result)
	}
	if strings.Contains(result, "Full plan output") {
		t.Fatalf("comment should omit raw plan output when includeDetails=false:\n%s", result)
	}
}

func TestBuildSummarySectionsWithOptions_WithoutDetailsClearsRowDetails(t *testing.T) {
	t.Parallel()

	plans := []ci.PlanResult{{
		ModuleID:          "svc/prod/us-east-1/vpc",
		Components:        map[string]string{"environment": "prod"},
		Status:            ci.PlanStatusChanges,
		Summary:           "+1",
		StructuredDetails: "### Resources\n- aws_vpc.main (create)",
		RawPlanOutput:     "+ resource \"aws_vpc\" \"main\"",
	}}

	sections, err := BuildSummarySectionsWithOptions(plans, nil, false)
	if err != nil {
		t.Fatalf("BuildSummarySectionsWithOptions() error = %v", err)
	}
	if len(sections) < 2 {
		t.Fatalf("sections = %#v, want module table row", sections)
	}
	rendered, err := ci.DecodeRenderSection(sections[1])
	if err != nil || len(rendered.Blocks) == 0 {
		t.Fatalf("RenderSection blocks = %v, err = %v", rendered.Blocks, err)
	}

	for _, block := range rendered.Blocks {
		if block.Kind == ci.RenderBlockKindDetails {
			t.Fatalf("unexpected details block when includeDetails=false: %+v", block)
		}
	}
}

func TestComposeComment_WithCostReport(t *testing.T) {
	t.Parallel()

	reports := []*ci.Report{{
		Producer: "cost",
		Title:    "Cost Estimation",
		Status:   ci.ReportStatusWarn,
		Sections: []ci.ReportSection{citest.MustRenderedSection(
			"Cost Estimation",
			"1 module, total: $15.00/mo (diff: +$5.00)",
			ci.ReportStatusWarn,
			ci.RenderTableBlock("", []string{"Module", "Before", "After", "Diff", "Notes"}, [][]string{{
				"svc/prod/us-east-1/vpc",
				"$10.00",
				"$15.00",
				"+$5.00",
				"-",
			}}),
		)},
	}}

	result := mustComposeComment(t, nil, reports, "", time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC))

	if !strings.Contains(result, "Cost Estimation") {
		t.Error("expected cost section")
	}
	if !strings.Contains(result, "$10.00") {
		t.Errorf("expected cost before in output, got: %s", result)
	}
}

func TestComposeCommentWithOptions_MalformedReportPayloadReturnsError(t *testing.T) {
	t.Parallel()

	_, err := ComposeCommentWithOptions(nil, []*ci.Report{{
		Producer: "policy",
		Title:    "Policy Check",
		Status:   ci.ReportStatusFail,
		Sections: []ci.ReportSection{{
			Kind:    ci.ReportSectionKindRendered,
			Title:   "Policy Check",
			Status:  ci.ReportStatusFail,
			Payload: []byte(`{"blocks":`),
		}},
	}}, "", "", time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC), true)
	if err == nil {
		t.Fatal("ComposeCommentWithOptions() error = nil, want malformed payload error")
	}
}

func TestComposeComment_EscapesMarkdownTableCells(t *testing.T) {
	t.Parallel()

	result := mustComposeComment(t, []ci.PlanResult{{
		ModuleID:   "svc/prod/us-east-1/vpc|main",
		Components: map[string]string{"environment": "prod"},
		Status:     ci.PlanStatusFailed,
		Error:      "bad | value\nnext line",
	}}, nil, "", time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC))

	if !strings.Contains(result, "vpc\\|main") {
		t.Fatalf("module path was not table-escaped:\n%s", result)
	}
	if !strings.Contains(result, "bad \\| value") {
		t.Fatalf("error text was not table-escaped:\n%s", result)
	}
	if !strings.Contains(result, "<br>") {
		t.Fatalf("newline was not converted in table cell:\n%s", result)
	}
}

func TestComposeComment_FiltersEnvironmentPlansToChangedAndFailed(t *testing.T) {
	t.Parallel()

	plans := []ci.PlanResult{
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

	result := mustComposeComment(t, plans, nil, "", time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC))

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
		Producer: "cost",
		Title:    "Cost Estimation",
		Status:   ci.ReportStatusWarn,
		Summary:  "3 modules, total: $27.00/mo (diff: +5.00)",
		Sections: []ci.ReportSection{citest.MustRenderedSection(
			"Cost Estimation",
			"3 modules, total: $27.00/mo (diff: +5.00)",
			ci.ReportStatusWarn,
			ci.RenderTableBlock("", []string{"Module", "Before", "After", "Diff", "Notes"}, [][]string{
				{"svc/prod/us-east-1/vpc", "$10.00", "$15.00", "+$5.00", "-"},
				{"svc/prod/us-east-1/redis", "$20.00", "$10.00", "-$10.00", "-"},
			}),
		)},
	}}

	result := mustComposeComment(t, nil, reports, "", time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC))

	if !strings.Contains(result, "svc/prod/us-east-1/vpc") {
		t.Fatalf("expected positive diff row in output: %s", result)
	}
	if !strings.Contains(result, "svc/prod/us-east-1/redis") {
		t.Fatalf("expected producer-rendered negative diff row in output: %s", result)
	}
}

func TestComposeComment_FiltersTfupdateReportToUpdatableModules(t *testing.T) {
	t.Parallel()

	reports := []*ci.Report{{
		Producer: "tfupdate",
		Title:    "Dependency Update Check",
		Status:   ci.ReportStatusWarn,
		Summary:  "4 checked, 2 updates available, 0 applied, 0 errors",
		Sections: []ci.ReportSection{citest.MustRenderedSection(
			"Dependency Update Check",
			"4 checked, 2 updates available, 0 applied, 0 errors",
			ci.ReportStatusWarn,
			ci.RenderTableBlock("Providers", []string{"Module", "Provider", "Current", "Latest", "Status"}, [][]string{{
				"svc/prod/us-east-1/vpc",
				"hashicorp/aws",
				"~> 5.0",
				"5.4.0",
				"update available",
			}}),
			ci.RenderTableBlock("Modules", []string{"Module", "Source", "Current", "Latest", "Status"}, [][]string{{
				"svc/prod/us-east-1/eks",
				"terraform-aws-modules/eks/aws",
				"20.0.0",
				"21.0.0",
				"applied",
			}}),
		)},
	}}

	result := mustComposeComment(t, nil, reports, "", time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC))

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

	result := mustComposeComment(t, []ci.PlanResult{}, nil, "abcdef1234567890abcdef1234567890abcdef12", time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC))

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
		plans []ci.PlanResult
		want  planStats
	}{
		{
			name:  "empty",
			plans: nil,
			want:  planStats{},
		},
		{
			name: "all statuses",
			plans: []ci.PlanResult{
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
			plans: []ci.PlanResult{
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
		plans    []ci.PlanResult
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
			plans: []ci.PlanResult{
				{Components: map[string]string{"environment": "prod"}},
				{Components: map[string]string{"environment": "prod"}},
				{Components: map[string]string{"environment": "staging"}},
			},
			wantKeys: []string{"prod", "staging"},
			wantLen:  map[string]int{"prod": 2, "staging": 1},
		},
		{
			name: "empty env becomes default",
			plans: []ci.PlanResult{
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
