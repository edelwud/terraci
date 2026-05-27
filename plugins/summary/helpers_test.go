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
	enabled          bool
	body             string
	currentBody      string
	currentFound     bool
	currentErr       error
	syncedPrevious   []string
	syncedCurrent    []string
	syncCalls        int
	syncErr          error
	calls            int
	err              error
	managedSupported bool
}

func (s *fakeCommentService) IsEnabled() bool { return s.enabled }

func (s *fakeCommentService) UpsertComment(_ context.Context, body string) error {
	s.calls++
	s.body = body
	return s.err
}

func (s *fakeCommentService) CurrentCommentBody(_ context.Context) (body string, found bool, err error) {
	if !s.managedSupported {
		return "", false, nil
	}
	return s.currentBody, s.currentFound, s.currentErr
}

func (s *fakeCommentService) SyncLabels(_ context.Context, previous, current []string) error {
	if !s.managedSupported {
		return nil
	}
	s.syncCalls++
	s.syncedPrevious = append([]string(nil), previous...)
	s.syncedCurrent = append([]string(nil), current...)
	return s.syncErr
}

type fakeSummaryProvider struct {
	commitSHA  string
	pipelineID string
	service    ci.CommentService
}

func (p *fakeSummaryProvider) CommitSHA() string  { return p.commitSHA }
func (p *fakeSummaryProvider) PipelineID() string { return p.pipelineID }
func (p *fakeSummaryProvider) CommentService() (ci.CommentService, bool) {
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

func newPlanReport(modulePath string, status ci.ReportStatus) *ci.Report {
	report, err := ci.NewRenderedReport(ci.RenderedReportOptions{
		Producer: "cost",
		Title:    "Cost Estimation",
		Status:   status,
		Summary:  "summary",
		Sections: []ci.RenderedSectionOptions{{
			Title:   "Cost Estimation",
			Summary: "summary",
			Blocks: []ci.RenderBlock{
				ci.NewTableBlock("", []ci.RenderColumn{
					ci.NewRenderColumn("Module"),
					ci.NewRenderColumn("Before"),
					ci.NewRenderColumn("After"),
					ci.NewRenderColumn("Diff"),
				}, []ci.RenderRow{
					ci.NewRenderRow(
						ci.RenderModulePath(modulePath),
						ci.RenderMoney(1, ci.RenderMoneyOptions{Unit: ci.RenderMoneyUnitMonth}),
						ci.RenderMoney(2, ci.RenderMoneyOptions{Unit: ci.RenderMoneyUnitMonth}),
						ci.RenderMoneyDelta(1, ci.RenderMoneyOptions{Unit: ci.RenderMoneyUnitMonth}),
					),
				}),
			},
		}},
	})
	if err != nil {
		panic(err)
	}
	return report
}
