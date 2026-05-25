package flow

import (
	"context"
	"fmt"

	log "github.com/caarlos0/log"

	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/execution"
	"github.com/edelwud/terraci/pkg/filter"
	"github.com/edelwud/terraci/pkg/pipeline"
	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/pkg/workflow"
	"github.com/edelwud/terraci/plugins/localexec/internal/render"
	"github.com/edelwud/terraci/plugins/localexec/internal/reports"
	"github.com/edelwud/terraci/plugins/localexec/internal/runner"
	"github.com/edelwud/terraci/plugins/localexec/internal/spec"
)

type Request = spec.Request

type Result struct {
	Execution     *execution.Result
	SummaryReport *ci.Report
	Skipped       bool
}

type UseCase struct {
	appCtx         *plugin.AppContext
	projects       ProjectPlanner
	irPlanner      IRPlanner
	contributions  ContributionCollector
	runtimeFactory runner.Factory
	summaryReports reports.Loader
}

type ProjectPlanner interface {
	Plan(ctx context.Context, req spec.Request) (*workflow.ProjectResult, error)
}

type IRPlanner interface {
	Build(project *workflow.ProjectResult, execCfg execution.Config, mode spec.ExecutionMode, contributions []*pipeline.Contribution) (*pipeline.IR, error)
}

type ContributionCollector interface {
	Collect(appCtx *plugin.AppContext) []*pipeline.Contribution
}

type Dependencies struct {
	Projects       ProjectPlanner
	IRPlanner      IRPlanner
	Contributions  ContributionCollector
	RuntimeFactory runner.Factory
	SummaryReports reports.Loader
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

func WithIRPlanner(builder IRPlanner) Option {
	return func(deps *Dependencies) {
		deps.IRPlanner = builder
	}
}

func WithContributionCollector(collector ContributionCollector) Option {
	return func(deps *Dependencies) {
		deps.Contributions = collector
	}
}

func WithSummaryReports(loader reports.Loader) Option {
	return func(deps *Dependencies) {
		deps.SummaryReports = loader
	}
}

func DefaultDependencies(appCtx *plugin.AppContext) Dependencies {
	structure := appCtx.Config().Structure()
	segments := append([]string(nil), structure.Segments...)
	return Dependencies{
		Projects:       newWorkflowProjectPlanner(appCtx),
		IRPlanner:      defaultIRPlanner{},
		Contributions:  contextContributionCollector{},
		RuntimeFactory: runner.NewFactory(),
		SummaryReports: reports.NewLoader(appCtx.Reports(), appCtx.WorkDir(), segments),
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
		irPlanner:      deps.IRPlanner,
		contributions:  deps.Contributions,
		runtimeFactory: deps.RuntimeFactory,
		summaryReports: deps.SummaryReports,
	}
}

func withDefaults(deps, defaults Dependencies) Dependencies {
	if deps.Projects == nil {
		deps.Projects = defaults.Projects
	}
	if deps.IRPlanner == nil {
		deps.IRPlanner = defaults.IRPlanner
	}
	if deps.Contributions == nil {
		deps.Contributions = defaults.Contributions
	}
	if deps.RuntimeFactory == nil {
		deps.RuntimeFactory = defaults.RuntimeFactory
	}
	if deps.SummaryReports == nil {
		deps.SummaryReports = defaults.SummaryReports
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
		return &Result{Skipped: true}, nil
	}

	execRuntime, err := u.runtimeFactory.Build(u.appCtx, runner.Options{Parallelism: req.Parallelism})
	if err != nil {
		return nil, err
	}

	contributions := u.contributions.Collect(u.appCtx)
	plan, err := u.irPlanner.Build(project, execRuntime.ExecConfig, req.Mode, contributions)
	if err != nil {
		return nil, err
	}
	reporter := render.NewProgressReporter()
	resultExec, err := execution.NewExecutor(
		execRuntime.JobRunner,
		execution.WithParallelism(execRuntime.ExecConfig.Parallelism),
		execution.WithEventSink(reporter),
	).Execute(ctx, plan)
	if err != nil {
		return &Result{Execution: resultExec}, err
	}

	summaryReport, err := u.summaryReports.Load(ctx)
	if err != nil {
		return nil, fmt.Errorf("load summary report: %w", err)
	}
	return &Result{Execution: resultExec, SummaryReport: summaryReport}, nil
}

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

type contextContributionCollector struct{}

func (contextContributionCollector) Collect(appCtx *plugin.AppContext) []*pipeline.Contribution {
	return appCtx.PipelineContributions()
}

type defaultIRPlanner struct{}

func (defaultIRPlanner) Build(project *workflow.ProjectResult, execCfg execution.Config, mode spec.ExecutionMode, contributions []*pipeline.Contribution) (*pipeline.IR, error) {
	planOnly := mode == spec.ExecutionModePlan
	if planOnly {
		execCfg.PlanEnabled = true
	}
	requirements := execCfg.BuildRequirements().Merge(pipeline.BuildRequirements{PlanOnly: planOnly})

	ir, err := pipeline.BuildProjectIR(pipeline.ProjectIRRequest{
		Project:       project,
		Script:        pipeline.ScriptConfig{InitEnabled: execCfg.InitEnabled, PlanEnabled: execCfg.PlanEnabled},
		Contributions: contributions,
		Requirements:  requirements,
		PlanEnabled:   execCfg.PlanEnabled,
	})
	if err != nil {
		return nil, fmt.Errorf("build local execution plan: %w", err)
	}
	return ir, nil
}
