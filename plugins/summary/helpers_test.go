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
func (p *fakeSummaryProvider) NewCommentService(_ *plugin.AppContext) ci.CommentService {
	return p.service
}

func newTestAppContext(t *testing.T, workDir string) *plugin.AppContext {
	t.Helper()
	return plugintest.NewAppContext(t, workDir)
}

func writePlanJSON(t *testing.T, workDir, modulePath, planJSON string) {
	t.Helper()
	if planJSON == "" {
		planJSON = testPlanWithChanges
	}

	moduleDir := filepath.Join(workDir, modulePath)
	if err := os.MkdirAll(moduleDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(moduleDir, "plan.json"), []byte(planJSON), 0o600); err != nil {
		t.Fatal(err)
	}
}

func writeReportJSON(t *testing.T, serviceDir, name string, report *ci.Report) {
	t.Helper()

	data, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(serviceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(serviceDir, name), data, 0o600); err != nil {
		t.Fatal(err)
	}
}

func newPlanReport(modulePath string, status ci.ReportStatus) *ci.Report {
	return &ci.Report{
		Plugin:  "cost",
		Title:   "Cost Estimation",
		Status:  status,
		Summary: "summary",
		Modules: []ci.ModuleReport{{
			ModulePath: modulePath,
			CostBefore: 1,
			CostAfter:  2,
			CostDiff:   1,
			HasCost:    true,
		}},
	}
}
