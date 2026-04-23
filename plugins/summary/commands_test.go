package summary

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/edelwud/terraci/pkg/ci"
	summaryengine "github.com/edelwud/terraci/plugins/summary/internal"
)

func TestPlugin_Commands_Registration(t *testing.T) {
	p := newTestPlugin()
	appCtx := newTestAppContext(t, t.TempDir())

	cmds := p.Commands(appCtx)
	if len(cmds) != 1 {
		t.Fatalf("Commands() returned %d commands, want 1", len(cmds))
	}
	if cmds[0].Use != "summary" {
		t.Fatalf("command.Use = %q, want summary", cmds[0].Use)
	}
}

func TestRunSummaryUseCase_NoPlanResults(t *testing.T) {
	appCtx := newTestAppContext(t, t.TempDir())

	output := plugSummaryOutput(t, func() {
		err := runSummaryUseCase(context.Background(), appCtx, &summaryengine.Config{}, func() (summaryProvider, error) {
			t.Fatal("provider resolver should not be called when no plan results exist")
			return nil, nil
		})
		if err != nil {
			t.Fatalf("runSummaryUseCase() error = %v", err)
		}
	})

	if !strings.Contains(output, "no plan results found") {
		t.Fatalf("output = %q, want no plan results warning", output)
	}
}

func TestRunSummaryUseCase_NoProvider_PrintsSummaryOnly(t *testing.T) {
	workDir := t.TempDir()
	appCtx := newTestAppContext(t, workDir)
	writePlanJSON(t, workDir, "platform/prod/us-east-1/vpc", testPlanWithChanges)

	output := plugSummaryOutput(t, func() {
		err := runSummaryUseCase(context.Background(), appCtx, &summaryengine.Config{}, func() (summaryProvider, error) {
			return nil, errors.New("no provider")
		})
		if err != nil {
			t.Fatalf("runSummaryUseCase() error = %v", err)
		}
	})

	if !strings.Contains(output, "no CI provider detected") {
		t.Fatalf("output = %q, want no provider message", output)
	}
	if !strings.Contains(output, "summary") {
		t.Fatalf("output = %q, want summary output", output)
	}

	report := readReportJSON(t, appCtx.ServiceDir(), "summary")
	if report.Plugin != "summary" {
		t.Fatalf("report plugin = %q, want summary", report.Plugin)
	}
	if report.Status != ci.ReportStatusWarn {
		t.Fatalf("report status = %q, want %q", report.Status, ci.ReportStatusWarn)
	}
}

func TestRunSummaryUseCase_PostsComment(t *testing.T) {
	workDir := t.TempDir()
	appCtx := newTestAppContext(t, workDir)
	modulePath := "platform/prod/us-east-1/vpc"
	writePlanJSON(t, workDir, modulePath, testPlanWithChanges)
	writeReportJSON(t, appCtx.ServiceDir(), "cost", newPlanReport(modulePath, ci.ReportStatusWarn))

	commentSvc := &fakeCommentService{enabled: true}
	provider := &fakeSummaryProvider{
		commitSHA:  "abcdef1234567890",
		pipelineID: "123",
		service:    commentSvc,
	}

	err := runSummaryUseCase(context.Background(), appCtx, &summaryengine.Config{}, func() (summaryProvider, error) {
		return provider, nil
	})
	if err != nil {
		t.Fatalf("runSummaryUseCase() error = %v", err)
	}
	if commentSvc.calls != 1 {
		t.Fatalf("comment upsert calls = %d, want 1", commentSvc.calls)
	}
	if !strings.Contains(commentSvc.body, "platform/prod/us-east-1/vpc") {
		t.Fatalf("comment body = %q, want module path", commentSvc.body)
	}
	if !strings.Contains(commentSvc.body, "terraci-plan-comment") {
		t.Fatalf("comment body = %q, want terraci marker", commentSvc.body)
	}

	report := readReportJSON(t, appCtx.ServiceDir(), "summary")
	if len(report.Sections) == 0 {
		t.Fatal("summary report sections are empty")
	}
	if report.Status != ci.ReportStatusWarn {
		t.Fatalf("report status = %q, want %q", report.Status, ci.ReportStatusWarn)
	}
}

func TestRunSummaryUseCase_OnChangesOnlySkipsNoChanges(t *testing.T) {
	workDir := t.TempDir()
	appCtx := newTestAppContext(t, workDir)
	writePlanJSON(t, workDir, "platform/prod/us-east-1/vpc", testPlanNoChanges)

	commentSvc := &fakeCommentService{enabled: true}
	output := plugSummaryOutput(t, func() {
		err := runSummaryUseCase(context.Background(), appCtx, &summaryengine.Config{OnChangesOnly: true}, func() (summaryProvider, error) {
			return &fakeSummaryProvider{
				commitSHA:  "abcdef1234567890",
				pipelineID: "123",
				service:    commentSvc,
			}, nil
		})
		if err != nil {
			t.Fatalf("runSummaryUseCase() error = %v", err)
		}
	})

	if commentSvc.calls != 0 {
		t.Fatalf("comment upsert calls = %d, want 0", commentSvc.calls)
	}
	if !strings.Contains(output, "no reportable changes") {
		t.Fatalf("output = %q, want no reportable changes message", output)
	}

	report := readReportJSON(t, appCtx.ServiceDir(), "summary")
	if report.Status != ci.ReportStatusPass {
		t.Fatalf("report status = %q, want %q", report.Status, ci.ReportStatusPass)
	}
}
