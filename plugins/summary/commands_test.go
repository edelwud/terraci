package summary

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/plugin"
	summaryengine "github.com/edelwud/terraci/plugins/summary/internal/summaryengine"
)

func TestPlugin_Commands_Registration(t *testing.T) {
	p := newTestPlugin()

	specs, err := p.CommandSpecs()
	if err != nil {
		t.Fatalf("CommandSpecs() error = %v", err)
	}
	if len(specs) != 1 {
		t.Fatalf("CommandSpecs() returned %d specs, want 1", len(specs))
	}
	cmd, err := plugin.BuildCommand(specs[0])
	if err != nil {
		t.Fatalf("BuildCommand() error = %v", err)
	}
	if cmd.Use != "summary" {
		t.Fatalf("command.Use = %q, want summary", cmd.Use)
	}
}

func TestRunSummaryUseCase_NoPlanResults(t *testing.T) {
	appCtx := newTestAppContext(t, t.TempDir())

	output := plugSummaryOutput(t, func() {
		err := runSummaryUseCase(context.Background(), appCtx, testSummaryRuntime(appCtx, &summaryengine.Config{}, func() (summaryengine.Provider, error) {
			t.Fatal("provider resolver should not be called when no plan results exist")
			return nil, nil
		}))
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
	writePlanJSON(t, workDir, testPlanWithChanges)

	output := plugSummaryOutput(t, func() {
		err := runSummaryUseCase(context.Background(), appCtx, testSummaryRuntime(appCtx, &summaryengine.Config{}, func() (summaryengine.Provider, error) {
			return nil, errors.New("no provider")
		}))
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
}

func TestRunSummaryUseCase_PostsComment(t *testing.T) {
	workDir := t.TempDir()
	appCtx := newTestAppContext(t, workDir)
	modulePath := testModulePath
	writePlanJSON(t, workDir, testPlanWithChanges)
	writeReportJSON(t, appCtx.ServiceDir(), "cost", newPlanReport(modulePath, ci.ReportStatusWarn))

	commentSvc := &fakeCommentService{enabled: true}
	provider := &fakeSummaryProvider{
		commitSHA:  "abcdef1234567890",
		pipelineID: "123",
		service:    commentSvc,
	}

	err := runSummaryUseCase(context.Background(), appCtx, testSummaryRuntime(appCtx, &summaryengine.Config{}, func() (summaryengine.Provider, error) {
		return provider, nil
	}))
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
}

func TestRunSummaryUseCase_OnChangesOnlySkipsNoChanges(t *testing.T) {
	workDir := t.TempDir()
	appCtx := newTestAppContext(t, workDir)
	writePlanJSON(t, workDir, testPlanNoChanges)

	commentSvc := &fakeCommentService{enabled: true}
	output := plugSummaryOutput(t, func() {
		err := runSummaryUseCase(context.Background(), appCtx, testSummaryRuntime(appCtx, &summaryengine.Config{OnChangesOnly: true}, func() (summaryengine.Provider, error) {
			return &fakeSummaryProvider{
				commitSHA:  "abcdef1234567890",
				pipelineID: "123",
				service:    commentSvc,
			}, nil
		}))
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
}

func TestRunSummaryUseCase_IncludeDetailsFalseRemovesDetailsFromCommentAndReport(t *testing.T) {
	workDir := t.TempDir()
	appCtx := newTestAppContext(t, workDir)
	writePlanJSON(t, workDir, testPlanWithChanges)

	commentSvc := &fakeCommentService{enabled: true}
	includeDetails := false
	err := runSummaryUseCase(context.Background(), appCtx, testSummaryRuntime(appCtx, &summaryengine.Config{IncludeDetails: &includeDetails}, func() (summaryengine.Provider, error) {
		return &fakeSummaryProvider{
			commitSHA:  "abcdef1234567890",
			pipelineID: "123",
			service:    commentSvc,
		}, nil
	}))
	if err != nil {
		t.Fatalf("runSummaryUseCase() error = %v", err)
	}

	if strings.Contains(commentSvc.body, "Full plan output") {
		t.Fatalf("comment body should omit full plan output when include_details=false:\n%s", commentSvc.body)
	}
}

func TestRunSummaryUseCase_EmbedsAndSyncsManagedLabels(t *testing.T) {
	workDir := t.TempDir()
	appCtx := newTestAppContext(t, workDir)
	writePlanJSON(t, workDir, testPlanWithChanges)

	commentSvc := &fakeCommentService{
		enabled:          true,
		managedSupported: true,
		currentFound:     true,
		currentBody:      ci.EmbedManagedLabels(ci.CommentMarker+"\n\nold", []string{"stale", "terraform"}),
	}
	cfg := &summaryengine.Config{Labels: []string{"terraform", "{environment}", "{module}", "resource:{resource_type}"}}
	err := runSummaryUseCase(context.Background(), appCtx, testSummaryRuntime(appCtx, cfg, func() (summaryengine.Provider, error) {
		return &fakeSummaryProvider{
			commitSHA:  "abcdef1234567890",
			pipelineID: "123",
			service:    commentSvc,
		}, nil
	}))
	if err != nil {
		t.Fatalf("runSummaryUseCase() error = %v", err)
	}

	wantLabels := []string{"prod", "resource:aws_instance", "terraform", "vpc"}
	if got := ci.ExtractManagedLabels(commentSvc.body); strings.Join(got, ",") != strings.Join(wantLabels, ",") {
		t.Fatalf("embedded labels = %v, want %v", got, wantLabels)
	}
	if commentSvc.syncCalls != 1 {
		t.Fatalf("label sync calls = %d, want 1", commentSvc.syncCalls)
	}
	if got := strings.Join(commentSvc.syncedPrevious, ","); got != "stale,terraform" {
		t.Fatalf("previous labels = %v, want [stale terraform]", commentSvc.syncedPrevious)
	}
	if got := strings.Join(commentSvc.syncedCurrent, ","); got != strings.Join(wantLabels, ",") {
		t.Fatalf("current labels = %v, want %v", commentSvc.syncedCurrent, wantLabels)
	}
}

func TestRunSummaryUseCase_LabelSyncErrorIsWarningOnly(t *testing.T) {
	workDir := t.TempDir()
	appCtx := newTestAppContext(t, workDir)
	writePlanJSON(t, workDir, testPlanWithChanges)

	commentSvc := &fakeCommentService{
		enabled:          true,
		managedSupported: true,
		syncErr:          errors.New("label api failed"),
	}
	err := runSummaryUseCase(context.Background(), appCtx, testSummaryRuntime(appCtx, &summaryengine.Config{Labels: []string{"terraform"}}, func() (summaryengine.Provider, error) {
		return &fakeSummaryProvider{
			commitSHA:  "abcdef1234567890",
			pipelineID: "123",
			service:    commentSvc,
		}, nil
	}))
	if err != nil {
		t.Fatalf("runSummaryUseCase() error = %v", err)
	}
	if commentSvc.calls != 1 {
		t.Fatalf("comment upsert calls = %d, want 1", commentSvc.calls)
	}
}

func testSummaryRuntime(appCtx *plugin.AppContext, cfg *summaryengine.Config, resolveProvider func() (summaryengine.Provider, error)) *summaryRuntime {
	runtime := newRuntime(appCtx, cfg)
	runtime.ProviderResolver = resolveProvider
	return runtime
}
