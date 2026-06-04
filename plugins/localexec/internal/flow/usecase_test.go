package flow

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/config"
	"github.com/edelwud/terraci/pkg/diagnostic"
	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/execution"
	"github.com/edelwud/terraci/pkg/graph"
	"github.com/edelwud/terraci/pkg/pipeline"
	"github.com/edelwud/terraci/pkg/pipeline/pipelinetest"
	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/pkg/plugin/plugintest"
	"github.com/edelwud/terraci/pkg/workflow"
	"github.com/edelwud/terraci/plugins/localexec/internal/reports"
	"github.com/edelwud/terraci/plugins/localexec/internal/runner"
	"github.com/edelwud/terraci/plugins/localexec/internal/spec"
)

type fakeProjectPlanner struct {
	project *workflow.ProjectResult
	err     error
}

func mustContribution(tb testing.TB, opts pipeline.ContributedJobOptions) *pipeline.Contribution {
	tb.Helper()
	job, err := pipeline.NewContributedJob(opts)
	if err != nil {
		tb.Fatalf("NewContributedJob() error = %v", err)
	}
	contribution, err := pipeline.NewContribution(job)
	if err != nil {
		tb.Fatalf("NewContribution() error = %v", err)
	}
	return contribution
}

func (p fakeProjectPlanner) Plan(context.Context, spec.Request) (*workflow.ProjectResult, error) {
	return p.project, p.err
}

type fakeRuntimeFactory struct {
	runtime *runner.Runtime
	err     error
	calls   int
	options []runner.RuntimeOptions
}

func (f *fakeRuntimeFactory) Build(opts runner.RuntimeOptions) (*runner.Runtime, error) {
	f.calls++
	f.options = append(f.options, opts)
	return f.runtime, f.err
}

type fakeJobRunner struct {
	mu   sync.Mutex
	jobs []string
	ran  []pipeline.Job
	err  error
}

func (r *fakeJobRunner) Run(_ context.Context, job pipeline.Job) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.jobs = append(r.jobs, job.Name())
	r.ran = append(r.ran, job)
	return r.err
}

func (r *fakeJobRunner) Jobs() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.jobs...)
}

func (r *fakeJobRunner) RanJobs() []pipeline.Job {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]pipeline.Job(nil), r.ran...)
}

type fakeEventSink struct {
	started  []string
	finished []string
}

func (s *fakeEventSink) JobStarted(event execution.JobEvent) {
	s.started = append(s.started, event.Name())
}

func (s *fakeEventSink) JobFinished(event execution.JobEvent, _ execution.JobResult) {
	s.finished = append(s.finished, event.Name())
}

type fakeSummaryReportLoader struct {
	result *reports.Result
	report *ci.Report
	err    error
	calls  int
}

func (l *fakeSummaryReportLoader) Load(context.Context) (*reports.Result, error) {
	l.calls++
	if l.result != nil || l.report == nil {
		return l.result, l.err
	}
	return reports.NewResult(l.report, diagnostic.List{}), l.err
}

func TestUseCase_RunUsesInjectedDependencies(t *testing.T) {
	workDir := t.TempDir()
	modulePath := filepath.Join(workDir, "platform", "stage", "eu-central-1", "vpc")
	if err := os.MkdirAll(modulePath, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(modulePath, "main.tf"), []byte("# test"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	appCtx := plugintest.NewAppContext(t, workDir)
	module := discovery.TestModule("platform", "stage", "eu-central-1", "vpc")
	jobRunner := &fakeJobRunner{}
	report := mustFlowReport(t, "summary", "Terraform Plan Summary")
	loader := &fakeSummaryReportLoader{report: report}
	eventSink := &fakeEventSink{}
	runtimeFactory := &fakeRuntimeFactory{runtime: &runner.Runtime{
		JobRunner: jobRunner,
	}}
	useCase := New(
		appCtx,
		WithProjectPlanner(fakeProjectWithTargets(module)),
		WithRuntimeFactory(runtimeFactory),
		WithSummaryReports(loader),
		WithEventSink(eventSink),
	)

	result, err := useCase.Run(context.Background(), spec.Request{})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if loader.calls != 1 {
		t.Fatalf("summary loader calls = %d, want 1", loader.calls)
	}
	if runtimeFactory.calls != 1 {
		t.Fatalf("runtime factory calls = %d, want 1", runtimeFactory.calls)
	}
	if got := runtimeFactory.options[0].PlanParallelism; got != config.DefaultConfig().Execution.Parallelism {
		t.Fatalf("runtime plan parallelism = %d, want config default", got)
	}
	if got := result.SummaryReport(); got == nil || got.Producer() != report.Producer() {
		t.Fatalf("summary report = %#v, want %#v", got, report)
	}
	if result.Execution() == nil || len(result.Execution().Groups()) == 0 {
		t.Fatal("expected execution groups to be recorded")
	}
	if len(eventSink.started) == 0 || len(eventSink.finished) == 0 {
		t.Fatalf("event sink did not receive execution events: started=%v finished=%v", eventSink.started, eventSink.finished)
	}
}

func mustFlowReport(tb testing.TB, producer, title string) *ci.Report {
	tb.Helper()
	report, err := ci.NewRenderedReport(ci.RenderedReportOptions{
		Producer: producer,
		Title:    title,
		Status:   ci.ReportStatusPass,
	})
	if err != nil {
		tb.Fatalf("NewRenderedReport() error = %v", err)
	}
	return report
}

func TestUseCase_RunBuildsIRFromProjectAndContributions(t *testing.T) {
	workDir := t.TempDir()
	modulePath := filepath.Join(workDir, "platform", "stage", "eu-central-1", "vpc")
	if err := os.MkdirAll(modulePath, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(modulePath, "main.tf"), []byte("# test"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	appCtx := plugintest.NewAppContext(t, workDir)
	module := discovery.TestModule("platform", "stage", "eu-central-1", "vpc")
	contributions := []*pipeline.Contribution{
		mustContribution(t, pipeline.ContributedJobOptions{
			Name:     "contributed",
			Commands: []string{"terraci contributed"},
		}),
	}
	loader := &fakeSummaryReportLoader{}
	jobRunner := &fakeJobRunner{}

	runtimeFactory := &fakeRuntimeFactory{runtime: &runner.Runtime{
		JobRunner: jobRunner,
	}}
	useCase := New(
		appCtx,
		WithProjectPlanner(fakeProjectWithTargets(module)),
		WithPipelineContributions(contributions),
		WithRuntimeFactory(runtimeFactory),
		WithSummaryReports(loader),
	)

	if _, err := useCase.Run(context.Background(), spec.Request{Mode: spec.ExecutionModePlan, Parallelism: 3}); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if got := runtimeFactory.options[0].PlanParallelism; got != 3 {
		t.Fatalf("runtime plan parallelism = %d, want request override 3", got)
	}
	executedJobs := jobRunner.Jobs()
	planJob := pipelinetest.MustJobByKind(t, pipelinetest.MustSingleModuleIR(t, module), pipeline.JobKindPlan)
	if !containsJob(executedJobs, planJob.Name()) {
		t.Fatalf("executed jobs = %#v, want module plan job", executedJobs)
	}
	if !containsJob(executedJobs, "contributed") {
		t.Fatalf("executed jobs = %#v, want contributed job", executedJobs)
	}
	if loader.calls != 1 {
		t.Fatalf("summary loader calls = %d, want 1", loader.calls)
	}
}

func TestUseCase_RunUsesConfigParallelismDefault(t *testing.T) {
	workDir, module := testWorkDirWithModule(t)
	cfg := config.DefaultConfig()
	cfg.Execution.Parallelism = 7
	appCtx := testAppContextWithConfig(workDir, cfg)
	runtimeFactory := &fakeRuntimeFactory{runtime: &runner.Runtime{JobRunner: &fakeJobRunner{}}}

	_, err := New(
		appCtx,
		WithProjectPlanner(fakeProjectWithTargets(module)),
		WithRuntimeFactory(runtimeFactory),
		WithSummaryReports(&fakeSummaryReportLoader{}),
	).Run(context.Background(), spec.Request{Mode: spec.ExecutionModePlan})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if got := runtimeFactory.options[0].PlanParallelism; got != 7 {
		t.Fatalf("runtime plan parallelism = %d, want config default 7", got)
	}
}

func TestUseCase_RunBuildsTerraformIntentFromConfig(t *testing.T) {
	workDir, module := testWorkDirWithModule(t)
	cfg := config.DefaultConfig()
	cfg.Execution.Binary = config.ExecutionBinaryTofu
	cfg.Execution.InitEnabled = false
	cfg.Execution.Env = map[string]string{
		"CUSTOM":    "value",
		"TF_MODULE": "should-not-win",
	}
	appCtx := testAppContextWithConfig(workDir, cfg)

	jobRunner := &fakeJobRunner{}
	_, err := New(
		appCtx,
		WithProjectPlanner(fakeProjectWithTargets(module)),
		WithPipelineContributions([]*pipeline.Contribution{
			mustContribution(t, testCommandJob("contributed")),
		}),
		WithRuntimeFactory(&fakeRuntimeFactory{runtime: &runner.Runtime{JobRunner: jobRunner}}),
		WithSummaryReports(&fakeSummaryReportLoader{}),
	).Run(context.Background(), spec.Request{Mode: spec.ExecutionModePlan})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	planJob := findRanJobByKind(t, jobRunner.RanJobs(), pipeline.JobKindPlan)
	terraformOp := planJob.Operation().Terraform()
	if terraformOp == nil {
		t.Fatal("plan job terraform operation = nil")
	}
	if terraformOp.Binary() != config.ExecutionBinaryTofu {
		t.Fatalf("Binary() = %q, want tofu", terraformOp.Binary())
	}
	if terraformOp.InitEnabled() {
		t.Fatal("InitEnabled() = true, want false")
	}
	if got := planJob.Env()["CUSTOM"]; got != "value" {
		t.Fatalf("plan job CUSTOM env = %q, want value", got)
	}
	if got := planJob.Env()["TF_MODULE"]; got != "vpc" {
		t.Fatalf("plan job TF_MODULE env = %q, want module-derived value", got)
	}

	commandJob := findRanJobByName(t, jobRunner.RanJobs(), "contributed")
	if _, ok := commandJob.Env()["CUSTOM"]; ok {
		t.Fatalf("command job env = %#v, should not inherit Terraform env", commandJob.Env())
	}
}

func TestUseCase_RunNoTargetsSkipsExecutionDependencies(t *testing.T) {
	workDir, module := testWorkDirWithModule(t)
	appCtx := plugintest.NewAppContext(t, workDir)
	runtimeFactory := &fakeRuntimeFactory{err: errors.New("runtime should not be built")}
	loader := &fakeSummaryReportLoader{err: errors.New("summary should not be loaded")}

	useCase := New(
		appCtx,
		WithProjectPlanner(fakeProjectPlanner{project: &workflow.ProjectResult{Workflow: fakeWorkflowResult(module)}}),
		WithRuntimeFactory(runtimeFactory),
		WithSummaryReports(loader),
	)

	result, err := useCase.Run(context.Background(), spec.Request{ModulePath: module.RelativePath})
	if err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}
	if result == nil || !result.Skipped() {
		t.Fatalf("Run() result = %#v, want skipped result", result)
	}

	if runtimeFactory.calls != 0 {
		t.Fatalf("runtime factory calls = %d, want 0", runtimeFactory.calls)
	}
	if loader.calls != 0 {
		t.Fatalf("summary loader calls = %d, want 0", loader.calls)
	}
}

func TestUseCase_RunReturnsTargetResolverError(t *testing.T) {
	workDir, _ := testWorkDirWithModule(t)
	appCtx := plugintest.NewAppContext(t, workDir)
	wantErr := errors.New("resolve targets")

	_, err := New(
		appCtx,
		WithProjectPlanner(fakeProjectPlanner{err: wantErr}),
		WithRuntimeFactory(&fakeRuntimeFactory{err: errors.New("runtime should not be built")}),
	).Run(context.Background(), spec.Request{})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Run() error = %v, want %v", err, wantErr)
	}
}

func TestUseCase_RunReturnsRuntimeFactoryError(t *testing.T) {
	workDir, module := testWorkDirWithModule(t)
	appCtx := plugintest.NewAppContext(t, workDir)
	wantErr := errors.New("build runtime")

	_, err := New(
		appCtx,
		WithProjectPlanner(fakeProjectWithTargets(module)),
		WithRuntimeFactory(&fakeRuntimeFactory{err: wantErr}),
	).Run(context.Background(), spec.Request{})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Run() error = %v, want %v", err, wantErr)
	}
}

func TestUseCase_RunReturnsTerraformProfileErrorBeforeRuntimeBuild(t *testing.T) {
	workDir, module := testWorkDirWithModule(t)
	cfg := config.DefaultConfig()
	cfg.Execution.Binary = "bad"
	appCtx := testAppContextWithConfig(workDir, cfg)
	runtimeFactory := &fakeRuntimeFactory{err: errors.New("runtime should not be built")}

	_, err := New(
		appCtx,
		WithProjectPlanner(fakeProjectWithTargets(module)),
		WithRuntimeFactory(runtimeFactory),
	).Run(context.Background(), spec.Request{})
	if err == nil || !strings.Contains(err.Error(), "unsupported terraform binary") {
		t.Fatalf("Run() error = %v, want unsupported terraform binary", err)
	}
	if runtimeFactory.calls != 0 {
		t.Fatalf("runtime factory calls = %d, want 0", runtimeFactory.calls)
	}
}

func TestUseCase_RunReturnsBuildProjectIRError(t *testing.T) {
	workDir, module := testWorkDirWithModule(t)
	appCtx := plugintest.NewAppContext(t, workDir)
	loader := &fakeSummaryReportLoader{}

	_, err := New(
		appCtx,
		WithProjectPlanner(fakeProjectPlanner{project: invalidProjectWithTargets(module)}),
		WithRuntimeFactory(&fakeRuntimeFactory{runtime: &runner.Runtime{
			JobRunner: &fakeJobRunner{},
		}}),
		WithSummaryReports(loader),
	).Run(context.Background(), spec.Request{})
	if err == nil || err.Error() != "build local execution plan: project workflow graph is required" {
		t.Fatalf("Run() error = %v, want invalid project graph", err)
	}
	if loader.calls != 0 {
		t.Fatalf("summary loader calls = %d, want 0", loader.calls)
	}
}

func TestUseCase_RunReturnsExecutionResultOnJobFailure(t *testing.T) {
	workDir, module := testWorkDirWithModule(t)
	appCtx := plugintest.NewAppContext(t, workDir)
	jobErr := errors.New("job failed")
	loader := &fakeSummaryReportLoader{err: errors.New("summary should not be loaded")}

	result, err := New(
		appCtx,
		WithProjectPlanner(fakeProjectWithTargets(module)),
		WithPipelineContributions([]*pipeline.Contribution{
			mustContribution(t, testCommandJob("summary")),
		}),
		WithRuntimeFactory(&fakeRuntimeFactory{runtime: &runner.Runtime{
			JobRunner: &fakeJobRunner{err: jobErr},
		}}),
		WithSummaryReports(loader),
	).Run(context.Background(), spec.Request{})
	if !errors.Is(err, jobErr) {
		t.Fatalf("Run() error = %v, want job failure %v", err, jobErr)
	}
	if result == nil || result.Execution() == nil {
		t.Fatalf("failure result = %#v, want failed job result", result)
	}
	if _, ok := result.Execution().Failed(); !ok {
		t.Fatalf("failure result = %#v, want failed job result", result)
	}
	if loader.calls != 0 {
		t.Fatalf("summary loader calls = %d, want 0", loader.calls)
	}
}

func TestUseCase_RunReturnsSummaryLoaderError(t *testing.T) {
	workDir := t.TempDir()
	modulePath := filepath.Join(workDir, "platform", "stage", "eu-central-1", "vpc")
	if err := os.MkdirAll(modulePath, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(modulePath, "main.tf"), []byte("# test"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	appCtx := plugintest.NewAppContext(t, workDir)
	module := discovery.TestModule("platform", "stage", "eu-central-1", "vpc")
	wantErr := errors.New("broken report")
	loader := &fakeSummaryReportLoader{err: wantErr}

	useCase := New(
		appCtx,
		WithProjectPlanner(fakeProjectWithTargets(module)),
		WithRuntimeFactory(&fakeRuntimeFactory{runtime: &runner.Runtime{
			JobRunner: &fakeJobRunner{},
		}}),
		WithSummaryReports(loader),
	)

	_, err := useCase.Run(context.Background(), spec.Request{})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Run() error = %v, want %v", err, wantErr)
	}

	if loader.calls != 1 {
		t.Fatalf("summary loader calls = %d, want 1", loader.calls)
	}
}

func TestUseCase_RunReturnsSummaryDiagnostics(t *testing.T) {
	workDir, module := testWorkDirWithModule(t)
	appCtx := plugintest.NewAppContext(t, workDir)
	diag := diagnostic.Warning("stale report skipped")

	result, err := New(
		appCtx,
		WithProjectPlanner(fakeProjectWithTargets(module)),
		WithRuntimeFactory(&fakeRuntimeFactory{runtime: &runner.Runtime{
			JobRunner: &fakeJobRunner{},
		}}),
		WithSummaryReports(&fakeSummaryReportLoader{result: reports.NewResult(nil, diagnostic.NewList(diag))}),
	).Run(context.Background(), spec.Request{})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if got := result.Diagnostics().Messages(); len(got) != 1 || got[0] != "stale report skipped" {
		t.Fatalf("Diagnostics = %v, want stale report warning", got)
	}
}

func TestNewRestoresDefaultsAfterNilOverrides(t *testing.T) {
	appCtx := plugintest.NewAppContext(t, t.TempDir())

	useCase := New(
		appCtx,
		WithProjectPlanner(nil),
		WithPipelineContributions(nil),
		WithRuntimeFactory(nil),
		WithSummaryReports(nil),
	)

	if useCase.projects == nil {
		t.Fatal("projects = nil, want default planner")
	}
	if useCase.contributions != nil {
		t.Fatalf("contributions = %#v, want none by default", useCase.contributions)
	}
	if useCase.runtimeFactory == nil {
		t.Fatal("runtimeFactory = nil, want default factory")
	}
	if useCase.summaryReports == nil {
		t.Fatal("summaryReports = nil, want default loader")
	}
	if useCase.eventSink == nil {
		t.Fatal("eventSink = nil, want default noop sink")
	}
}

var _ ProjectPlanner = fakeProjectPlanner{}
var _ runner.Factory = (*fakeRuntimeFactory)(nil)
var _ reports.Loader = (*fakeSummaryReportLoader)(nil)

func testWorkDirWithModule(t *testing.T) (string, *discovery.Module) {
	t.Helper()

	workDir := t.TempDir()
	module := discovery.TestModule("platform", "stage", "eu-central-1", "vpc")
	modulePath := filepath.Join(workDir, module.RelativePath)
	if err := os.MkdirAll(modulePath, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(modulePath, "main.tf"), []byte("# test"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	return workDir, module
}

func fakeProjectWithTargets(targets ...*discovery.Module) ProjectPlanner {
	return fakeProjectPlanner{project: &workflow.ProjectResult{
		Workflow: fakeWorkflowResult(targets...),
		Targets:  targets,
	}}
}

func invalidProjectWithTargets(targets ...*discovery.Module) *workflow.ProjectResult {
	return &workflow.ProjectResult{
		Workflow: &workflow.Result{
			Filtered: workflow.NewModuleSet(targets),
		},
		Targets: targets,
	}
}

func testAppContextWithConfig(workDir string, cfg *config.Config) *plugin.AppContext {
	return plugin.NewAppContext(plugin.AppContextOptions{Config: cfg, WorkDir: workDir})
}

func findRanJobByKind(tb testing.TB, jobs []pipeline.Job, kind pipeline.JobKind) pipeline.Job {
	tb.Helper()
	for i := range jobs {
		job := jobs[i]
		if job.Kind() == kind {
			return job
		}
	}
	tb.Fatalf("job kind %q not found in %v", kind, jobNames(jobs))
	return jobs[0]
}

func findRanJobByName(tb testing.TB, jobs []pipeline.Job, name string) pipeline.Job {
	tb.Helper()
	for i := range jobs {
		job := jobs[i]
		if job.Name() == name {
			return job
		}
	}
	tb.Fatalf("job %q not found in %v", name, jobNames(jobs))
	return jobs[0]
}

func jobNames(jobs []pipeline.Job) []string {
	names := make([]string, 0, len(jobs))
	for i := range jobs {
		names = append(names, jobs[i].Name())
	}
	return names
}

func fakeWorkflowResult(modules ...*discovery.Module) *workflow.Result {
	return &workflow.Result{
		All:      workflow.NewModuleSet(modules),
		Filtered: workflow.NewModuleSet(modules),
		Graph:    graph.BuildFromDependencies(modules, nil),
	}
}

func testCommandJob(name string) pipeline.ContributedJobOptions {
	return pipeline.ContributedJobOptions{Name: name, Commands: []string{"true"}}
}

func containsJob(jobs []string, name string) bool {
	return slices.Contains(jobs, name)
}
