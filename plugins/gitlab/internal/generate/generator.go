package generate

import (
	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/execution"
	"github.com/edelwud/terraci/pkg/graph"
	"github.com/edelwud/terraci/pkg/pipeline"
	configpkg "github.com/edelwud/terraci/plugins/gitlab/internal/config"
	"github.com/edelwud/terraci/plugins/gitlab/internal/domain"
)

const (
	DefaultStagesPrefix = "deploy"
	WhenManual          = "manual"
)

// Generator transforms TerraCi IR into GitLab CI domain models.
type Generator struct {
	settings          settings
	stagePlanner      stagePlanner
	jobBuilder        jobBuilder
	contributionIndex contributionIndex
	contributions     []*pipeline.Contribution
	depGraph          *graph.DependencyGraph
	modules           []*discovery.Module
	moduleIndex       *discovery.ModuleIndex
}

// NewGenerator creates a new pipeline generator.
func NewGenerator(cfg *configpkg.Config, execCfg execution.Config, contributions []*pipeline.Contribution, depGraph *graph.DependencyGraph, modules []*discovery.Module) *Generator {
	cfgSettings := newSettings(cfg, execCfg)
	index := newContributionIndex(contributions)
	return &Generator{
		settings:     cfgSettings,
		stagePlanner: newStagePlanner(cfgSettings, index),
		jobBuilder: newJobBuilder(cfgSettings, index, func(job *domain.Job, jobType configpkg.JobOverwriteType) {
			applyResolvedJobConfig(cfgSettings, job, jobType)
		}),
		contributionIndex: index,
		contributions:     contributions,
		depGraph:          depGraph,
		modules:           modules,
		moduleIndex:       discovery.NewModuleIndex(modules),
	}
}

// Generate creates a GitLab CI pipeline for the given modules.
func (g *Generator) Generate(targetModules []*discovery.Module) (pipeline.GeneratedPipeline, error) {
	ir, err := g.buildIR(targetModules)
	if err != nil {
		return nil, err
	}

	return g.transform(ir), nil
}

func (g *Generator) buildIR(targetModules []*discovery.Module) (*pipeline.IR, error) {
	return pipeline.Build(pipeline.BuildOptions{
		DepGraph:      g.depGraph,
		TargetModules: targetModules,
		AllModules:    g.modules,
		ModuleIndex:   g.moduleIndex,
		Script: pipeline.ScriptConfig{
			InitEnabled:  g.settings.initEnabled(),
			DetailedPlan: g.IsMREnabled(),
			PlanEnabled:  g.settings.planEnabled(),
			AutoApprove:  g.settings.autoApprove(),
		},
		Contributions: g.contributions,
		PlanEnabled:   g.settings.planEnabled(),
		PlanOnly:      g.settings.planOnly(),
	})
}

func (g *Generator) transform(ir *pipeline.IR) *domain.Pipeline {
	effectiveImage := g.settings.defaultImage()

	result := &domain.Pipeline{
		Stages:    g.stagePlanner.stages(ir),
		Variables: g.settings.variables(),
		Default: &domain.DefaultConfig{
			Image: &domain.ImageConfig{
				Name:       effectiveImage.Name,
				Entrypoint: effectiveImage.Entrypoint,
			},
		},
		Jobs:     make(map[string]*domain.Job),
		Workflow: g.generateWorkflow(),
	}

	prefix := g.stagesPrefix()
	for _, level := range ir.Levels {
		for _, mj := range level.Modules {
			if mj.Plan != nil {
				result.Jobs[mj.Plan.Name] = g.jobBuilder.planJob(mj.Plan, mj.Module, level.Index, prefix)
			}
			if mj.Apply != nil {
				result.Jobs[mj.Apply.Name] = g.jobBuilder.applyJob(mj.Apply, mj.Module, level.Index, prefix)
			}
		}
	}

	if g.contributionIndex.hasContributedJobs() {
		for i := range ir.Jobs {
			cj := &ir.Jobs[i]
			job := g.jobBuilder.contributedJob(cj)
			if cj.Phase == pipeline.PhaseFinalize {
				g.jobBuilder.applySummaryOverrides(job)
			}
			result.Jobs[cj.Name] = job
		}
	}

	return result
}

func (g *Generator) DryRun(targetModules []*discovery.Module) (*pipeline.DryRunResult, error) {
	ir, err := g.buildIR(targetModules)
	if err != nil {
		return nil, err
	}

	result := pipeline.BuildDryRunResult(ir, len(g.modules))
	result.Stages = len(g.stagePlanner.stages(ir))
	return result, nil
}

// IsMREnabled returns true if MR integration is enabled in config.
func (g *Generator) IsMREnabled() bool {
	return g.settings.mrCommentEnabled()
}

func (g *Generator) stagesPrefix() string {
	return g.settings.stagesPrefix()
}
