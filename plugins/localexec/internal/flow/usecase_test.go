package flow

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"testing"

	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/diagnostic"
	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/execution"
	"github.com/edelwud/terraci/pkg/graph"
	"github.com/edelwud/terraci/pkg/pipeline"
	"github.com/edelwud/terraci/pkg/pipeline/pipelinetest"
	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/pkg/plugin/plugintest"
	"github.com/edelwud/terraci/pkg/terraformrun"
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

func mustProfile(tb testing.TB, opts terraformrun.ProfileOptions) terraformrun.Profile {
	tb.Helper()
	profile, err := terraformrun.NewProfile(opts)
	if err != nil {
		tb.Fatalf("NewProfile() error = %v", err)
	}
	return profile
}

func (p fakeProjectPlanner) Plan(context.Context, spec.Request) (*workflow.ProjectResult, error) {
	return p.project, p.err
}

type fakeRuntimeFactory struct {
	runtime *runner.Runtime
	err     error
	calls   int
}

func (f *fakeRuntimeFactory) Build(*plugin.AppContext, runner.Options) (*runner.Runtime, error) {
	f.calls++
	return f.runtime, f.err
}

type fakeJobRunner struct {
	mu   sync.Mutex
	jobs []string
	err  error
}

func (r *fakeJobRunner) Run(_ context.Context, job pipeline.Job) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.jobs = append(r.jobs, job.Name())
	return r.err
}

func (r *fakeJobRunner) Jobs() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.jobs...)
}

type fakeContributionCollector struct {
	contributions []*pipeline.Contribution
	calls         int
}

func (c *fakeContributionCollector) Collect(*plugin.AppContext) []*pipeline.Contribution {
	c.calls++
	return c.contributions
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
	report := &ci.Report{Producer: "summary", Title: "Terraform Plan Summary"}
	loader := &fakeSummaryReportLoader{report: report}
	eventSink := &fakeEventSink{}
	runtimeFactory := &fakeRuntimeFactory{runtime: &runner.Runtime{
		Profile:   mustProfile(t, terraformrun.ProfileOptions{Parallelism: 1}),
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
	if got := result.SummaryReport(); got == nil || got.Producer != report.Producer {
		t.Fatalf("summary report = %#v, want %#v", got, report)
	}
	if result.Execution() == nil || len(result.Execution().Groups()) == 0 {
		t.Fatal("expected execution groups to be recorded")
	}
	if len(eventSink.started) == 0 || len(eventSink.finished) == 0 {
		t.Fatalf("event sink did not receive execution events: started=%v finished=%v", eventSink.started, eventSink.finished)
	}
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
	contributionCollector := &fakeContributionCollector{
		contributions: []*pipeline.Contribution{mustContribution(t, pipeline.ContributedJobOptions{
			Name:     "contributed",
			Commands: []string{"terraci contributed"},
		})},
	}
	loader := &fakeSummaryReportLoader{}
	jobRunner := &fakeJobRunner{}

	runtimeFactory := &fakeRuntimeFactory{runtime: &runner.Runtime{
		Profile:   mustProfile(t, terraformrun.ProfileOptions{Parallelism: 3}),
		JobRunner: jobRunner,
	}}
	useCase := New(
		appCtx,
		WithProjectPlanner(fakeProjectWithTargets(module)),
		WithContributionCollector(contributionCollector),
		WithRuntimeFactory(runtimeFactory),
		WithSummaryReports(loader),
	)

	if _, err := useCase.Run(context.Background(), spec.Request{Mode: spec.ExecutionModePlan}); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if contributionCollector.calls != 1 {
		t.Fatalf("contribution collector calls = %d, want 1", contributionCollector.calls)
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

func TestUseCase_RunReturnsBuildProjectIRError(t *testing.T) {
	workDir, module := testWorkDirWithModule(t)
	appCtx := plugintest.NewAppContext(t, workDir)
	loader := &fakeSummaryReportLoader{}

	_, err := New(
		appCtx,
		WithProjectPlanner(fakeProjectPlanner{project: invalidProjectWithTargets(module)}),
		WithRuntimeFactory(&fakeRuntimeFactory{runtime: &runner.Runtime{
			Profile:   mustProfile(t, terraformrun.ProfileOptions{Parallelism: 1}),
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
		WithContributionCollector(&fakeContributionCollector{contributions: []*pipeline.Contribution{
			mustContribution(t, testCommandJob("summary")),
		}}),
		WithRuntimeFactory(&fakeRuntimeFactory{runtime: &runner.Runtime{
			Profile:   mustProfile(t, terraformrun.ProfileOptions{Parallelism: 1}),
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
			Profile:   mustProfile(t, terraformrun.ProfileOptions{Parallelism: 1}),
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
			Profile:   mustProfile(t, terraformrun.ProfileOptions{Parallelism: 1}),
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
		WithContributionCollector(nil),
		WithRuntimeFactory(nil),
		WithSummaryReports(nil),
	)

	if useCase.projects == nil {
		t.Fatal("projects = nil, want default planner")
	}
	if useCase.contributions == nil {
		t.Fatal("contributions = nil, want default collector")
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
