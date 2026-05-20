package flow

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/execution"
	"github.com/edelwud/terraci/pkg/pipeline"
	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/pkg/plugin/plugintest"
	"github.com/edelwud/terraci/pkg/workflow"
	"github.com/edelwud/terraci/plugins/localexec/internal/planner"
	"github.com/edelwud/terraci/plugins/localexec/internal/reports"
	"github.com/edelwud/terraci/plugins/localexec/internal/runner"
	"github.com/edelwud/terraci/plugins/localexec/internal/spec"
	"github.com/edelwud/terraci/plugins/localexec/internal/targeting"
)

type fakeTargetResolver struct {
	targets []*discovery.Module
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

func (r fakeTargetResolver) Resolve(context.Context, spec.Request, *workflow.Result) ([]*discovery.Module, error) {
	return r.targets, r.err
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
	jobs []string
	err  error
}

func (r *fakeJobRunner) Run(_ context.Context, job *pipeline.Job) error {
	r.jobs = append(r.jobs, job.Name)
	return r.err
}

type fakePlanner struct {
	plan          *pipeline.IR
	err           error
	calls         int
	targets       []*discovery.Module
	mode          spec.ExecutionMode
	parallelism   int
	planEnabled   bool
	filteredCount int
	contributions []*pipeline.Contribution
}

func (p *fakePlanner) Build(targets []*discovery.Module, result *workflow.Result, execCfg execution.Config, mode spec.ExecutionMode, contributions []*pipeline.Contribution) (*pipeline.IR, error) {
	p.calls++
	p.targets = targets
	p.mode = mode
	p.parallelism = execCfg.Parallelism
	p.planEnabled = execCfg.PlanEnabled
	p.contributions = contributions
	if result != nil {
		p.filteredCount = len(result.Filtered.Modules)
	}
	return p.plan, p.err
}

type fakeContributionCollector struct {
	contributions []*pipeline.Contribution
	calls         int
}

func (c *fakeContributionCollector) Collect(*plugin.AppContext) []*pipeline.Contribution {
	c.calls++
	return c.contributions
}

type fakeSummaryReportLoader struct {
	report *ci.Report
	err    error
	calls  int
}

func (l *fakeSummaryReportLoader) Load(context.Context) (*ci.Report, error) {
	l.calls++
	return l.report, l.err
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
	runtimeFactory := &fakeRuntimeFactory{runtime: &runner.Runtime{
		ExecConfig: execution.Config{PlanEnabled: true, Parallelism: 1},
		JobRunner:  jobRunner,
	}}
	useCase := New(
		appCtx,
		WithTargetResolver(fakeTargetResolver{targets: []*discovery.Module{module}}),
		WithRuntimeFactory(runtimeFactory),
		WithSummaryReports(loader),
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
	if result.SummaryReport != report {
		t.Fatalf("summary report = %#v, want %#v", result.SummaryReport, report)
	}
	if result.Execution == nil || len(result.Execution.Groups) == 0 {
		t.Fatal("expected execution groups to be recorded")
	}
}

func TestUseCase_RunUsesInjectedPlanner(t *testing.T) {
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
	ir := &pipeline.IR{
		Jobs: []pipeline.Job{testCommandJob("summary")},
	}
	plannerStub := &fakePlanner{plan: ir}
	contributionCollector := &fakeContributionCollector{
		contributions: []*pipeline.Contribution{mustContribution(t, pipeline.ContributedJobOptions{
			Name:     "contributed",
			Commands: []string{"terraci contributed"},
		})},
	}
	loader := &fakeSummaryReportLoader{}

	runtimeFactory := &fakeRuntimeFactory{runtime: &runner.Runtime{
		ExecConfig: execution.Config{PlanEnabled: true, Parallelism: 3},
		JobRunner:  &fakeJobRunner{},
	}}
	useCase := New(
		appCtx,
		WithTargetResolver(fakeTargetResolver{targets: []*discovery.Module{module}}),
		WithPlanner(plannerStub),
		WithContributionCollector(contributionCollector),
		WithRuntimeFactory(runtimeFactory),
		WithSummaryReports(loader),
	)

	if _, err := useCase.Run(context.Background(), spec.Request{Mode: spec.ExecutionModePlan}); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if plannerStub.calls != 1 {
		t.Fatalf("planner calls = %d, want 1", plannerStub.calls)
	}
	if plannerStub.mode != spec.ExecutionModePlan {
		t.Fatalf("planner mode = %v, want %v", plannerStub.mode, spec.ExecutionModePlan)
	}
	if plannerStub.parallelism != 3 {
		t.Fatalf("planner parallelism = %d, want 3", plannerStub.parallelism)
	}
	if !plannerStub.planEnabled {
		t.Fatal("planner exec config should preserve plan_enabled")
	}
	if len(plannerStub.targets) != 1 || plannerStub.targets[0].ID() != module.ID() {
		t.Fatalf("planner targets = %#v, want module %q", plannerStub.targets, module.ID())
	}
	if plannerStub.filteredCount != 1 {
		t.Fatalf("planner filtered count = %d, want 1", plannerStub.filteredCount)
	}
	if contributionCollector.calls != 1 {
		t.Fatalf("contribution collector calls = %d, want 1", contributionCollector.calls)
	}
	if len(plannerStub.contributions) != 1 {
		t.Fatalf("planner contributions = %#v, want one contribution", plannerStub.contributions)
	}
	jobs := plannerStub.contributions[0].Jobs()
	if len(jobs) != 1 || jobs[0].Name() != "contributed" {
		t.Fatalf("planner contributions = %#v, want contributed job", plannerStub.contributions)
	}
	if loader.calls != 1 {
		t.Fatalf("summary loader calls = %d, want 1", loader.calls)
	}
}

func TestUseCase_RunNoTargetsSkipsExecutionDependencies(t *testing.T) {
	workDir, module := testWorkDirWithModule(t)
	appCtx := plugintest.NewAppContext(t, workDir)
	runtimeFactory := &fakeRuntimeFactory{err: errors.New("runtime should not be built")}
	plannerStub := &fakePlanner{err: errors.New("planner should not be called")}
	loader := &fakeSummaryReportLoader{err: errors.New("summary should not be loaded")}

	useCase := New(
		appCtx,
		WithTargetResolver(fakeTargetResolver{targets: nil}),
		WithPlanner(plannerStub),
		WithRuntimeFactory(runtimeFactory),
		WithSummaryReports(loader),
	)

	result, err := useCase.Run(context.Background(), spec.Request{ModulePath: module.RelativePath})
	if err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}
	if result == nil || !result.Skipped {
		t.Fatalf("Run() result = %#v, want skipped result", result)
	}

	if runtimeFactory.calls != 0 {
		t.Fatalf("runtime factory calls = %d, want 0", runtimeFactory.calls)
	}
	if plannerStub.calls != 0 {
		t.Fatalf("planner calls = %d, want 0", plannerStub.calls)
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
		WithTargetResolver(fakeTargetResolver{err: wantErr}),
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
	plannerStub := &fakePlanner{err: errors.New("planner should not be called")}

	_, err := New(
		appCtx,
		WithTargetResolver(fakeTargetResolver{targets: []*discovery.Module{module}}),
		WithPlanner(plannerStub),
		WithRuntimeFactory(&fakeRuntimeFactory{err: wantErr}),
	).Run(context.Background(), spec.Request{})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Run() error = %v, want %v", err, wantErr)
	}
	if plannerStub.calls != 0 {
		t.Fatalf("planner calls = %d, want 0", plannerStub.calls)
	}
}

func TestUseCase_RunReturnsPlannerError(t *testing.T) {
	workDir, module := testWorkDirWithModule(t)
	appCtx := plugintest.NewAppContext(t, workDir)
	wantErr := errors.New("build plan")
	loader := &fakeSummaryReportLoader{}

	_, err := New(
		appCtx,
		WithTargetResolver(fakeTargetResolver{targets: []*discovery.Module{module}}),
		WithPlanner(&fakePlanner{err: wantErr}),
		WithRuntimeFactory(&fakeRuntimeFactory{runtime: &runner.Runtime{
			ExecConfig: execution.Config{PlanEnabled: true, Parallelism: 1},
			JobRunner:  &fakeJobRunner{},
		}}),
		WithSummaryReports(loader),
	).Run(context.Background(), spec.Request{})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Run() error = %v, want %v", err, wantErr)
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
		WithTargetResolver(fakeTargetResolver{targets: []*discovery.Module{module}}),
		WithPlanner(&fakePlanner{plan: &pipeline.IR{
			Jobs: []pipeline.Job{testCommandJob("summary")},
		}}),
		WithRuntimeFactory(&fakeRuntimeFactory{runtime: &runner.Runtime{
			ExecConfig: execution.Config{PlanEnabled: true, Parallelism: 1},
			JobRunner:  &fakeJobRunner{err: jobErr},
		}}),
		WithSummaryReports(loader),
	).Run(context.Background(), spec.Request{})
	if !errors.Is(err, jobErr) {
		t.Fatalf("Run() error = %v, want job failure %v", err, jobErr)
	}
	if result == nil || result.Execution == nil || result.Execution.Failed() == nil {
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
		WithTargetResolver(fakeTargetResolver{targets: []*discovery.Module{module}}),
		WithRuntimeFactory(&fakeRuntimeFactory{runtime: &runner.Runtime{
			ExecConfig: execution.Config{PlanEnabled: true, Parallelism: 1},
			JobRunner:  &fakeJobRunner{},
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

func TestNewRestoresDefaultsAfterNilOverrides(t *testing.T) {
	appCtx := plugintest.NewAppContext(t, t.TempDir())

	useCase := New(
		appCtx,
		WithTargetResolver(nil),
		WithPlanner(nil),
		WithContributionCollector(nil),
		WithRuntimeFactory(nil),
		WithSummaryReports(nil),
	)

	if useCase.targets == nil {
		t.Fatal("targets = nil, want default resolver")
	}
	if useCase.planner == nil {
		t.Fatal("planner = nil, want default builder")
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
}

var _ targeting.Resolver = fakeTargetResolver{}
var _ planner.Builder = (*fakePlanner)(nil)
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

func testCommandJob(name string) pipeline.Job {
	return pipeline.Job{
		Name: name,
		Kind: pipeline.JobKindCommand,
		Operation: pipeline.Operation{
			Type:     pipeline.OperationTypeCommands,
			Commands: []string{"true"},
		},
	}
}
