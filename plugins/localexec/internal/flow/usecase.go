package flow

import (
	"context"

	"github.com/edelwud/terraci/pkg/execution"
	"github.com/edelwud/terraci/pkg/filter"
	"github.com/edelwud/terraci/pkg/log"
	"github.com/edelwud/terraci/pkg/pipeline"
	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/pkg/workflow"
	"github.com/edelwud/terraci/plugins/localexec/internal/planner"
	"github.com/edelwud/terraci/plugins/localexec/internal/render"
	"github.com/edelwud/terraci/plugins/localexec/internal/runner"
	"github.com/edelwud/terraci/plugins/localexec/internal/spec"
	"github.com/edelwud/terraci/plugins/localexec/internal/targeting"
)

type UseCase struct {
	appCtx         *plugin.AppContext
	targets        targeting.Resolver
	planner        planner.Builder
	contributions  ContributionCollector
	runtimeFactory runner.Factory
	summaryReports render.SummaryReportLoader
	output         render.Output
}

type ContributionCollector interface {
	Collect(appCtx *plugin.AppContext) []*pipeline.Contribution
}

type Dependencies struct {
	Targets        targeting.Resolver
	Planner        planner.Builder
	Contributions  ContributionCollector
	RuntimeFactory runner.Factory
	SummaryReports render.SummaryReportLoader
	Output         render.Output
}

type Option func(*Dependencies)

func WithTargetResolver(resolver targeting.Resolver) Option {
	return func(deps *Dependencies) {
		deps.Targets = resolver
	}
}

func WithRuntimeFactory(factory runner.Factory) Option {
	return func(deps *Dependencies) {
		deps.RuntimeFactory = factory
	}
}

func WithPlanner(builder planner.Builder) Option {
	return func(deps *Dependencies) {
		deps.Planner = builder
	}
}

func WithContributionCollector(collector ContributionCollector) Option {
	return func(deps *Dependencies) {
		deps.Contributions = collector
	}
}

func WithOutput(output render.Output) Option {
	return func(deps *Dependencies) {
		deps.Output = output
	}
}

func WithSummaryReports(loader render.SummaryReportLoader) Option {
	return func(deps *Dependencies) {
		deps.SummaryReports = loader
	}
}

func DefaultDependencies(appCtx *plugin.AppContext) Dependencies {
	segments := []string(nil)
	if cfg := appCtx.Config(); cfg != nil {
		segments = append(segments, cfg.Structure.Segments...)
	}
	return Dependencies{
		Targets:        targeting.NewWorkflowResolver(appCtx, nil),
		Planner:        planner.New(),
		Contributions:  contextContributionCollector{},
		RuntimeFactory: runner.NewFactory(),
		SummaryReports: render.NewSummaryReportLoader(appCtx.ServiceDir(), appCtx.WorkDir(), segments),
		Output:         render.NewLogOutput(),
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
		targets:        deps.Targets,
		planner:        deps.Planner,
		contributions:  deps.Contributions,
		runtimeFactory: deps.RuntimeFactory,
		summaryReports: deps.SummaryReports,
		output:         deps.Output,
	}
}

func withDefaults(deps, defaults Dependencies) Dependencies {
	if deps.Targets == nil {
		deps.Targets = defaults.Targets
	}
	if deps.Planner == nil {
		deps.Planner = defaults.Planner
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
	if deps.Output == nil {
		deps.Output = defaults.Output
	}
	return deps
}

func (u *UseCase) Run(ctx context.Context, req spec.ExecuteRequest) error {
	result, err := workflow.Run(ctx, workflowOptionsFromContext(u.appCtx, req.Filters))
	if err != nil {
		return err
	}

	targets, err := u.targets.Resolve(ctx, req, result)
	if err != nil {
		return err
	}
	if len(targets) == 0 {
		log.Info("no modules to process")
		return nil
	}

	execRuntime, err := u.runtimeFactory.Build(u.appCtx, runner.Options{Parallelism: req.Parallelism})
	if err != nil {
		return err
	}

	contributions := u.contributions.Collect(u.appCtx)
	plan, err := u.planner.Build(targets, result, execRuntime.ExecConfig, req.Mode, contributions)
	if err != nil {
		return err
	}
	if resetErr := u.summaryReports.Reset(); resetErr != nil {
		return resetErr
	}

	reporter := render.NewProgressReporter()
	resultExec, err := execution.NewExecutor(
		execRuntime.JobRunner,
		execution.WithParallelism(execRuntime.ExecConfig.Parallelism),
		execution.WithEventSink(reporter),
	).Execute(ctx, plan)
	if err != nil {
		return u.output.Failure(resultExec, err)
	}

	summaryReport, err := u.summaryReports.Load()
	if err != nil {
		log.WithError(err).Warn("skip summary report rendering")
	}
	u.output.Completed(resultExec, summaryReport)
	return nil
}

func workflowOptionsFromContext(appCtx *plugin.AppContext, ff *filter.Flags) workflow.Options {
	return workflow.OptionsFromConfig(appCtx.WorkDir(), appCtx.Config(), ff)
}

type contextContributionCollector struct{}

func (contextContributionCollector) Collect(appCtx *plugin.AppContext) []*pipeline.Contribution {
	return plugin.CollectContributions(appCtx)
}
