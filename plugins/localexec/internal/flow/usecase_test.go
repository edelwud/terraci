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
	"github.com/edelwud/terraci/plugins/localexec/internal/render"
	"github.com/edelwud/terraci/plugins/localexec/internal/runner"
	"github.com/edelwud/terraci/plugins/localexec/internal/spec"
	"github.com/edelwud/terraci/plugins/localexec/internal/targeting"
)

type fakeTargetResolver struct {
	targets []*discovery.Module
	err     error
}

func (r fakeTargetResolver) Resolve(context.Context, spec.ExecuteRequest, *workflow.Result) ([]*discovery.Module, error) {
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

type fakeOutput struct {
	completedCalls int
	failureCalls   int
	result         *execution.Result
	failureResult  *execution.Result
	failureErr     error
	failureReturn  error
	summaryReport  *ci.Report
}

func (o *fakeOutput) Completed(result *execution.Result, summaryReport *ci.Report) {
	o.completedCalls++
	o.result = result
	o.summaryReport = summaryReport
}

func (o *fakeOutput) Failure(result *execution.Result, err error) error {
	o.failureCalls++
	o.failureResult = result
	o.failureErr = err
	if o.failureReturn != nil {
		return o.failureReturn
	}
	return err
}

type fakeSummaryReportLoader struct {
	report     *ci.Report
	err        error
	resetErr   error
	calls      int
	resetCalls int
}

func (l *fakeSummaryReportLoader) Reset() error {
	l.resetCalls++
	return l.resetErr
}

func (l *fakeSummaryReportLoader) Load() (*ci.Report, error) {
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
	output := &fakeOutput{}
	report := &ci.Report{Producer: ci.AggregateReportProducer, Title: "Terraform Plan Summary"}
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
		WithOutput(output),
	)

	if err := useCase.Run(context.Background(), spec.ExecuteRequest{}); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if output.completedCalls != 1 {
		t.Fatalf("completed calls = %d, want 1", output.completedCalls)
	}
	if loader.calls != 1 {
		t.Fatalf("summary loader calls = %d, want 1", loader.calls)
	}
	if loader.resetCalls != 1 {
		t.Fatalf("summary reset calls = %d, want 1", loader.resetCalls)
	}
	if runtimeFactory.calls != 1 {
		t.Fatalf("runtime factory calls = %d, want 1", runtimeFactory.calls)
	}
	if output.summaryReport != report {
		t.Fatalf("summary report = %#v, want %#v", output.summaryReport, report)
	}
	if len(output.result.Groups) == 0 {
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
		Jobs: []pipeline.Job{{Name: "summary", Phase: pipeline.PhaseFinalize}},
	}
	plannerStub := &fakePlanner{plan: ir}
	contributionCollector := &fakeContributionCollector{
		contributions: []*pipeline.Contribution{{Jobs: []pipeline.ContributedJob{{Name: "contributed"}}}},
	}
	output := &fakeOutput{}
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
		WithOutput(output),
	)

	if err := useCase.Run(context.Background(), spec.ExecuteRequest{Mode: spec.ExecutionModePlan}); err != nil {
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
	if len(plannerStub.contributions) != 1 || plannerStub.contributions[0].Jobs[0].Name != "contributed" {
		t.Fatalf("planner contributions = %#v, want contributed job", plannerStub.contributions)
	}
	if loader.calls != 1 {
		t.Fatalf("summary loader calls = %d, want 1", loader.calls)
	}
	if loader.resetCalls != 1 {
		t.Fatalf("summary reset calls = %d, want 1", loader.resetCalls)
	}
}

func TestUseCase_RunNoTargetsSkipsExecutionDependencies(t *testing.T) {
	workDir, module := testWorkDirWithModule(t)
	appCtx := plugintest.NewAppContext(t, workDir)
	runtimeFactory := &fakeRuntimeFactory{err: errors.New("runtime should not be built")}
	plannerStub := &fakePlanner{err: errors.New("planner should not be called")}
	loader := &fakeSummaryReportLoader{err: errors.New("summary should not be loaded")}
	output := &fakeOutput{}

	useCase := New(
		appCtx,
		WithTargetResolver(fakeTargetResolver{targets: nil}),
		WithPlanner(plannerStub),
		WithRuntimeFactory(runtimeFactory),
		WithSummaryReports(loader),
		WithOutput(output),
	)

	if err := useCase.Run(context.Background(), spec.ExecuteRequest{ModulePath: module.RelativePath}); err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
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
	if loader.resetCalls != 0 {
		t.Fatalf("summary reset calls = %d, want 0", loader.resetCalls)
	}
	if output.completedCalls != 0 || output.failureCalls != 0 {
		t.Fatalf("output calls completed=%d failure=%d, want none", output.completedCalls, output.failureCalls)
	}
}

func TestUseCase_RunReturnsTargetResolverError(t *testing.T) {
	workDir, _ := testWorkDirWithModule(t)
	appCtx := plugintest.NewAppContext(t, workDir)
	wantErr := errors.New("resolve targets")

	err := New(
		appCtx,
		WithTargetResolver(fakeTargetResolver{err: wantErr}),
		WithRuntimeFactory(&fakeRuntimeFactory{err: errors.New("runtime should not be built")}),
		WithOutput(&fakeOutput{}),
	).Run(context.Background(), spec.ExecuteRequest{})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Run() error = %v, want %v", err, wantErr)
	}
}

func TestUseCase_RunReturnsRuntimeFactoryError(t *testing.T) {
	workDir, module := testWorkDirWithModule(t)
	appCtx := plugintest.NewAppContext(t, workDir)
	wantErr := errors.New("build runtime")
	plannerStub := &fakePlanner{err: errors.New("planner should not be called")}

	err := New(
		appCtx,
		WithTargetResolver(fakeTargetResolver{targets: []*discovery.Module{module}}),
		WithPlanner(plannerStub),
		WithRuntimeFactory(&fakeRuntimeFactory{err: wantErr}),
		WithOutput(&fakeOutput{}),
	).Run(context.Background(), spec.ExecuteRequest{})
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
	output := &fakeOutput{}

	err := New(
		appCtx,
		WithTargetResolver(fakeTargetResolver{targets: []*discovery.Module{module}}),
		WithPlanner(&fakePlanner{err: wantErr}),
		WithRuntimeFactory(&fakeRuntimeFactory{runtime: &runner.Runtime{
			ExecConfig: execution.Config{PlanEnabled: true, Parallelism: 1},
			JobRunner:  &fakeJobRunner{},
		}}),
		WithSummaryReports(loader),
		WithOutput(output),
	).Run(context.Background(), spec.ExecuteRequest{})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Run() error = %v, want %v", err, wantErr)
	}
	if loader.calls != 0 {
		t.Fatalf("summary loader calls = %d, want 0", loader.calls)
	}
	if loader.resetCalls != 0 {
		t.Fatalf("summary reset calls = %d, want 0", loader.resetCalls)
	}
	if output.completedCalls != 0 || output.failureCalls != 0 {
		t.Fatalf("output calls completed=%d failure=%d, want none", output.completedCalls, output.failureCalls)
	}
}

func TestUseCase_RunJobFailureGoesThroughOutputFailure(t *testing.T) {
	workDir, module := testWorkDirWithModule(t)
	appCtx := plugintest.NewAppContext(t, workDir)
	jobErr := errors.New("job failed")
	outputErr := errors.New("render failure")
	output := &fakeOutput{failureReturn: outputErr}
	loader := &fakeSummaryReportLoader{err: errors.New("summary should not be loaded")}

	err := New(
		appCtx,
		WithTargetResolver(fakeTargetResolver{targets: []*discovery.Module{module}}),
		WithPlanner(&fakePlanner{plan: &pipeline.IR{
			Jobs: []pipeline.Job{{Name: "summary", Phase: pipeline.PhaseFinalize}},
		}}),
		WithRuntimeFactory(&fakeRuntimeFactory{runtime: &runner.Runtime{
			ExecConfig: execution.Config{PlanEnabled: true, Parallelism: 1},
			JobRunner:  &fakeJobRunner{err: jobErr},
		}}),
		WithSummaryReports(loader),
		WithOutput(output),
	).Run(context.Background(), spec.ExecuteRequest{})
	if !errors.Is(err, outputErr) {
		t.Fatalf("Run() error = %v, want output failure %v", err, outputErr)
	}
	if output.failureCalls != 1 {
		t.Fatalf("failure calls = %d, want 1", output.failureCalls)
	}
	if !errors.Is(output.failureErr, jobErr) {
		t.Fatalf("output failure err = %v, want %v", output.failureErr, jobErr)
	}
	if output.failureResult == nil || output.failureResult.Failed() == nil {
		t.Fatalf("failure result = %#v, want failed job result", output.failureResult)
	}
	if output.completedCalls != 0 {
		t.Fatalf("completed calls = %d, want 0", output.completedCalls)
	}
	if loader.calls != 0 {
		t.Fatalf("summary loader calls = %d, want 0", loader.calls)
	}
	if loader.resetCalls != 1 {
		t.Fatalf("summary reset calls = %d, want 1", loader.resetCalls)
	}
}

func TestUseCase_RunIgnoresSummaryLoaderError(t *testing.T) {
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
	output := &fakeOutput{}
	loader := &fakeSummaryReportLoader{err: errors.New("broken report")}

	useCase := New(
		appCtx,
		WithTargetResolver(fakeTargetResolver{targets: []*discovery.Module{module}}),
		WithRuntimeFactory(&fakeRuntimeFactory{runtime: &runner.Runtime{
			ExecConfig: execution.Config{PlanEnabled: true, Parallelism: 1},
			JobRunner:  &fakeJobRunner{},
		}}),
		WithSummaryReports(loader),
		WithOutput(output),
	)

	if err := useCase.Run(context.Background(), spec.ExecuteRequest{}); err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}

	if loader.calls != 1 {
		t.Fatalf("summary loader calls = %d, want 1", loader.calls)
	}
	if loader.resetCalls != 1 {
		t.Fatalf("summary reset calls = %d, want 1", loader.resetCalls)
	}
	if output.completedCalls != 1 {
		t.Fatalf("completed calls = %d, want 1", output.completedCalls)
	}
	if output.summaryReport != nil {
		t.Fatalf("summary report = %#v, want nil after loader error", output.summaryReport)
	}
}

func TestUseCase_RunReturnsSummaryResetErrorBeforeExecution(t *testing.T) {
	workDir, module := testWorkDirWithModule(t)
	appCtx := plugintest.NewAppContext(t, workDir)
	wantErr := errors.New("reset summary")
	loader := &fakeSummaryReportLoader{resetErr: wantErr}
	output := &fakeOutput{}
	jobRunner := &fakeJobRunner{}

	err := New(
		appCtx,
		WithTargetResolver(fakeTargetResolver{targets: []*discovery.Module{module}}),
		WithRuntimeFactory(&fakeRuntimeFactory{runtime: &runner.Runtime{
			ExecConfig: execution.Config{PlanEnabled: true, Parallelism: 1},
			JobRunner:  jobRunner,
		}}),
		WithSummaryReports(loader),
		WithOutput(output),
	).Run(context.Background(), spec.ExecuteRequest{})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Run() error = %v, want %v", err, wantErr)
	}
	if len(jobRunner.jobs) != 0 {
		t.Fatalf("executed jobs = %v, want none", jobRunner.jobs)
	}
	if loader.calls != 0 {
		t.Fatalf("summary loader calls = %d, want 0", loader.calls)
	}
	if loader.resetCalls != 1 {
		t.Fatalf("summary reset calls = %d, want 1", loader.resetCalls)
	}
	if output.completedCalls != 0 || output.failureCalls != 0 {
		t.Fatalf("output calls completed=%d failure=%d, want none", output.completedCalls, output.failureCalls)
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
		WithOutput(nil),
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
	if useCase.output == nil {
		t.Fatal("output = nil, want default output")
	}
}

var _ targeting.Resolver = fakeTargetResolver{}
var _ planner.Builder = (*fakePlanner)(nil)
var _ runner.Factory = (*fakeRuntimeFactory)(nil)
var _ render.SummaryReportLoader = (*fakeSummaryReportLoader)(nil)
var _ render.Output = (*fakeOutput)(nil)

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
