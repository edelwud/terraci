package summary

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/pkg/plugin/plugintest"
)

const testPlanWithChanges = `{
	"format_version": "1.2",
	"terraform_version": "1.6.0",
	"resource_changes": [{
		"address": "aws_instance.web",
		"module_address": "",
		"type": "aws_instance",
		"name": "web",
		"change": {
			"actions": ["create"],
			"before": null,
			"after": {"instance_type": "t3.micro", "ami": "ami-12345"},
			"after_unknown": {}
		}
	}]
}`

const testPlanNoChanges = `{
	"format_version": "1.2",
	"terraform_version": "1.6.0",
	"resource_changes": []
}`

const testModulePath = "platform/prod/us-east-1/vpc"

type fakeCommentService struct {
	enabled bool
	body    string
	calls   int
	err     error
}

func (s *fakeCommentService) IsEnabled() bool { return s.enabled }

func (s *fakeCommentService) UpsertComment(_ context.Context, body string) error {
	s.calls++
	s.body = body
	return s.err
}

type fakeSummaryProvider struct {
	commitSHA  string
	pipelineID string
	service    ci.CommentService
}

func (p *fakeSummaryProvider) CommitSHA() string  { return p.commitSHA }
func (p *fakeSummaryProvider) PipelineID() string { return p.pipelineID }
func (p *fakeSummaryProvider) NewCommentService(_ *plugin.AppContext) (ci.CommentService, bool) {
	return p.service, p.service != nil
}

func newTestAppContext(t *testing.T, workDir string) *plugin.AppContext {
	t.Helper()
	return plugintest.NewAppContext(t, workDir)
}

func writePlanJSON(t *testing.T, workDir, planJSON string) {
	t.Helper()
	if planJSON == "" {
		planJSON = testPlanWithChanges
	}

	moduleDir := filepath.Join(workDir, testModulePath)
	if err := os.MkdirAll(moduleDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(moduleDir, "plan.json"), []byte(planJSON), 0o600); err != nil {
		t.Fatal(err)
	}
}

func writeReportJSON(t *testing.T, serviceDir, pluginName string, report *ci.Report) {
	t.Helper()

	data, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(serviceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(serviceDir, ci.ReportFilename(pluginName)), data, 0o600); err != nil {
		t.Fatal(err)
	}
}

func readSummaryReportJSON(t *testing.T, serviceDir string) *ci.Report {
	t.Helper()

	report, err := ci.LoadReport(filepath.Join(serviceDir, ci.ReportFilename("summary")))
	if err != nil {
		t.Fatal(err)
	}

	return report
}

func newPlanReport(modulePath string, status ci.ReportStatus) *ci.Report {
	payload, err := json.Marshal(map[string]any{
		"totals": map[string]any{"after": 2, "diff": 1},
		"rows": []map[string]any{{
			"module_path": modulePath,
			"before":      1,
			"after":       2,
			"diff":        1,
			"has_cost":    true,
		}},
	})
	if err != nil {
		panic(err)
	}

	return &ci.Report{
		Producer: "cost",
		Title:    "Cost Estimation",
		Status:   status,
		Summary:  "summary",
		Sections: []ci.ReportSection{{
			Kind:           "cost_changes",
			Title:          "Cost Estimation",
			Status:         status,
			SectionSummary: "summary",
			Payload:        payload,
		}},
	}
}
