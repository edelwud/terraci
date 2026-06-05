package flow

import (
	"context"
	"fmt"

	log "github.com/caarlos0/log"

	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/diagnostic"
	"github.com/edelwud/terraci/pkg/execution"
	"github.com/edelwud/terraci/pkg/filter"
	"github.com/edelwud/terraci/pkg/pipeline"
	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/pkg/terraformrun"
	"github.com/edelwud/terraci/pkg/workflow"
	"github.com/edelwud/terraci/plugins/localexec/internal/reports"
	"github.com/edelwud/terraci/plugins/localexec/internal/runner"
	"github.com/edelwud/terraci/plugins/localexec/internal/spec"
)

type Request = spec.Request

type Result struct {
	execution     *execution.Result
	summaryReport *ci.Report
	skipped       bool
	diagnostics   diagnostic.List
}

func skippedResult() *Result {
	return &Result{skipped: true}
}

func completedResult(exec *execution.Result, report *ci.Report, diagnostics diagnostic.List) *Result {
	return &Result{
		execution:     exec.Clone(),
		summaryReport: report.Clone(),
		diagnostics:   diagnostics,
	}
}

func (r *Result) Execution() *execution.Result {
	if r == nil {
		return nil
	}
	return r.execution.Clone()
}

func (r *Result) SummaryReport() *ci.Report {
	if r == nil {
		return nil
	}
	return r.summaryReport.Clone()
}

func (r *Result) Skipped() bool {
	return r != nil && r.skipped
}

func (r *Result) Diagnostics() diagnostic.List {
	if r == nil {
		return diagnostic.List{}
	}
	return r.diagnostics
}

type UseCase struct {
	appCtx         *plugin.AppContext
	projects       ProjectPlanner
	contributions  pipeline.ContributionSet
	runtimeFactory runner.Factory
	summaryReports reports.Loader
	eventSink      execution.EventSink
}

type ProjectPlanner interface {
	Plan(ctx context.Context, req spec.Request) (*workflow.ProjectResult, error)
}

type Dependencies struct {
	Projects       ProjectPlanner
	Contributions  pipeline.ContributionSet
	RuntimeFactory runner.Factory
	SummaryReports reports.Loader
	EventSink      execution.EventSink
}

type Option func(*Dependencies)

func WithProjectPlanner(projects ProjectPlanner) Option {
	return func(deps *Dependencies) {
		deps.Projects = projects
	}
}

func WithRuntimeFactory(factory runner.Factory) Option {
	return func(deps *Dependencies) {
		deps.RuntimeFactory = factory
	}
}

func WithPipelineContributions(contributions pipeline.ContributionSet) Option {
	return func(deps *Dependencies) {
		deps.Contributions = contributions.Clone()
	}
}

func WithSummaryReports(loader reports.Loader) Option {
	return func(deps *Dependencies) {
		deps.SummaryReports = loader
	}
}

func WithEventSink(sink execution.EventSink) Option {
	return func(deps *Dependencies) {
		deps.EventSink = sink
	}
}

func DefaultDependencies(appCtx *plugin.AppContext) Dependencies {
	structure := appCtx.Config().Structure()
	segments := append([]string(nil), structure.Segments...)
	return Dependencies{
		Projects:       newWorkflowProjectPlanner(appCtx),
		RuntimeFactory: runner.NewFactory(),
		SummaryReports: reports.NewLoader(appCtx.Reports(), appCtx.WorkDir(), segments),
		EventSink:      noopEventSink{},
	}
}

func New(appCtx *plugin.AppContext, opts ...Option) *UseCase {
	defaults := DefaultDependencies(appCtx)
	deps := defaults
	for _, opt := range opts {
		if opt != nil {
			opt(&deps)
		}
	}
	deps = withDefaults(deps, defaults)

	return &UseCase{
		appCtx:         appCtx,
		projects:       deps.Projects,
		contributions:  deps.Contributions.Clone(),
		runtimeFactory: deps.RuntimeFactory,
		summaryReports: deps.SummaryReports,
		eventSink:      deps.EventSink,
	}
}

func withDefaults(deps, defaults Dependencies) Dependencies {
	if deps.Projects == nil {
		deps.Projects = defaults.Projects
	}
	if deps.RuntimeFactory == nil {
		deps.RuntimeFactory = defaults.RuntimeFactory
	}
	if deps.SummaryReports == nil {
		deps.SummaryReports = defaults.SummaryReports
	}
	if deps.EventSink == nil {
		deps.EventSink = defaults.EventSink
	}
	return deps
}

func (u *UseCase) Run(ctx context.Context, req Request) (*Result, error) {
	project, err := u.projects.Plan(ctx, req)
	if err != nil {
		return nil, err
	}

	if len(project.Targets) == 0 {
		log.Info("no modules to process")
		return skippedResult(), nil
	}

	profile, err := profileForRequest(u.appCtx, req)
	if err != nil {
		return nil, fmt.Errorf("terraform profile: %w", err)
	}

	plan, err := buildExecutionIR(project, profile, req.Mode, u.contributions)
	if err != nil {
		return nil, err
	}

	execRuntime, err := u.runtimeFactory.Build(runner.RuntimeOptions{
		WorkDir:         u.appCtx.WorkDir(),
		ServiceDir:      u.appCtx.ServiceDir(),
		PlanParallelism: profile.Parallelism(),
	})
	if err != nil {
		return nil, err
	}

	resultExec, err := execution.NewExecutor(
		execRuntime.JobRunner,
		execution.WithParallelism(profile.Parallelism()),
		execution.WithEventSink(u.eventSink),
	).Execute(ctx, plan)
	if err != nil {
		return completedResult(resultExec, nil, diagnostic.List{}), err
	}

	summaryResult, err := u.summaryReports.Load(ctx)
	if err != nil {
		return completedResult(resultExec, nil, diagnostic.List{}), fmt.Errorf("load summary report: %w", err)
	}
	var (
		summaryReport *ci.Report
		diagnostics   diagnostic.List
	)
	if summaryResult != nil {
		summaryReport = summaryResult.Report()
		diagnostics = summaryResult.Diagnostics()
	}
	return completedResult(resultExec, summaryReport, diagnostics), nil
}

type noopEventSink struct{}

func (noopEventSink) JobStarted(execution.JobEvent) {}

func (noopEventSink) JobFinished(execution.JobEvent, execution.JobResult) {}

type workflowProjectPlanner struct {
	appCtx *plugin.AppContext
}

func newWorkflowProjectPlanner(appCtx *plugin.AppContext) ProjectPlanner {
	return workflowProjectPlanner{appCtx: appCtx}
}

func (p workflowProjectPlanner) Plan(ctx context.Context, req spec.Request) (*workflow.ProjectResult, error) {
	return workflow.PlanProject(ctx, workflow.ProjectRequest{
		WorkDir: p.appCtx.WorkDir(),
		Config:  p.appCtx.Config(),
		Filters: derefFilters(req.Filters),
		Targeting: workflow.TargetRequest{
			Enabled:     true,
			ModulePath:  req.ModulePath,
			ChangedOnly: req.ChangedOnly,
			BaseRef:     req.BaseRef,
			ChangeDetectorResolver: func() (workflow.ChangeDetector, error) {
				return p.appCtx.ChangeDetectorResolver().ResolveChangeDetector()
			},
		},
	})
}

func derefFilters(filters *filter.Flags) filter.Flags {
	if filters == nil {
		return filter.Flags{}
	}
	return *filters
}

func profileForRequest(appCtx *plugin.AppContext, req Request) (terraformrun.Profile, error) {
	profile, err := terraformrun.ProfileFromConfig(appCtx.Config())
	if err != nil {
		return terraformrun.Profile{}, err
	}
	if req.Parallelism > 0 {
		return profile.WithParallelism(req.Parallelism)
	}
	return profile, nil
}

func buildExecutionIR(project *workflow.ProjectResult, profile terraformrun.Profile, mode spec.ExecutionMode, contributions pipeline.ContributionSet) (*pipeline.IR, error) {
	intent, err := intentForMode(mode)
	if err != nil {
		return nil, fmt.Errorf("build local execution intent: %w", err)
	}
	terraformConfig, err := pipeline.NewTerraformJobConfigFromProfile(profile)
	if err != nil {
		return nil, fmt.Errorf("terraform job config: %w", err)
	}

	ir, err := pipeline.BuildProjectIR(pipeline.ProjectIRRequest{
		Project:       project,
		Terraform:     terraformConfig,
		Contributions: contributions,
		Intent:        intent,
	})
	if err != nil {
		return nil, fmt.Errorf("build local execution plan: %w", err)
	}
	return ir, nil
}

func intentForMode(mode spec.ExecutionMode) (pipeline.BuildIntent, error) {
	switch mode {
	case spec.ExecutionModeRun:
		return pipeline.ApplyBuildIntent()
	case spec.ExecutionModePlan:
		return pipeline.PlanBuildIntent(pipeline.AllPlanResources(pipeline.ResourceKindPlanBinary))
	default:
		var zero pipeline.BuildIntent
		return zero, fmt.Errorf("unsupported local execution mode %q", mode)
	}
}
