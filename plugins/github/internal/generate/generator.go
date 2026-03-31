package generate

import (
	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/graph"
	"github.com/edelwud/terraci/pkg/pipeline"
	configpkg "github.com/edelwud/terraci/plugins/github/internal/config"
	domainpkg "github.com/edelwud/terraci/plugins/github/internal/domain"
)

type Generator struct {
	settings      settings
	contributions []*pipeline.Contribution
	depGraph      *graph.DependencyGraph
	modules       []*discovery.Module
	moduleIndex   *discovery.ModuleIndex
}

func NewGenerator(cfg *configpkg.Config, contributions []*pipeline.Contribution, depGraph *graph.DependencyGraph, modules []*discovery.Module) *Generator {
	return &Generator{
		settings:      newSettings(cfg),
		contributions: contributions,
		depGraph:      depGraph,
		modules:       modules,
		moduleIndex:   discovery.NewModuleIndex(modules),
	}
}

func (g *Generator) Generate(targetModules []*discovery.Module) (pipeline.GeneratedPipeline, error) {
	ir, err := g.buildIR(targetModules)
	if err != nil {
		return nil, err
	}

	return g.transform(ir, targetModules), nil
}

func (g *Generator) DryRun(targetModules []*discovery.Module) (*pipeline.DryRunResult, error) {
	plan, err := pipeline.BuildJobPlan(
		g.depGraph,
		targetModules,
		g.modules,
		g.moduleIndex,
		g.hasContributedJobs(),
		g.settings.planEnabled(),
	)
	if err != nil {
		return nil, err
	}

	return pipeline.BuildDryRunResult(plan, len(g.modules), g.settings.planEnabled()), nil
}

func (g *Generator) IsPREnabled() bool {
	return g.settings.prEnabled()
}

func (g *Generator) buildIR(targetModules []*discovery.Module) (*pipeline.IR, error) {
	return pipeline.Build(pipeline.BuildOptions{
		DepGraph:      g.depGraph,
		TargetModules: targetModules,
		AllModules:    g.modules,
		ModuleIndex:   g.moduleIndex,
		Script: pipeline.ScriptConfig{
			TerraformBinary: g.settings.terraformBinary(),
			InitEnabled:     g.settings.initEnabled(),
			PlanEnabled:     g.settings.planEnabled(),
			AutoApprove:     g.settings.autoApprove(),
			DetailedPlan:    g.settings.prEnabled(),
		},
		Contributions: g.contributions,
		PlanEnabled:   g.settings.planEnabled(),
		PlanOnly:      g.settings.planOnly(),
	})
}

func (g *Generator) transform(ir *pipeline.IR, targetModules []*discovery.Module) *domainpkg.Workflow {
	workflow := newWorkflowBuilder(g.settings).baseWorkflow()
	targetSet := g.buildTargetSet(targetModules)
	builder := newJobBuilder(g.settings, targetSet, g.depGraph, g.moduleIndex)

	for _, level := range ir.Levels {
		for _, moduleJobs := range level.Modules {
			if moduleJobs.Plan != nil {
				workflow.Jobs[moduleJobs.Plan.Name] = builder.planJob(moduleJobs.Plan, moduleJobs.Module)
			}
			if moduleJobs.Apply != nil {
				workflow.Jobs[moduleJobs.Apply.Name] = builder.applyJob(moduleJobs.Apply, moduleJobs.Module)
			}
		}
	}

	for i := range ir.Jobs {
		job := builder.contributedJob(&ir.Jobs[i])
		workflow.Jobs[ir.Jobs[i].Name] = job
	}

	return workflow
}

func (g *Generator) buildTargetSet(targetModules []*discovery.Module) map[string]bool {
	if len(targetModules) == 0 {
		targetModules = g.modules
	}

	targetSet := make(map[string]bool, len(targetModules))
	for _, module := range targetModules {
		targetSet[module.ID()] = true
	}
	return targetSet
}

func (g *Generator) hasContributedJobs() bool {
	for _, contribution := range g.contributions {
		if len(contribution.Jobs) > 0 {
			return true
		}
	}
	return false
}
