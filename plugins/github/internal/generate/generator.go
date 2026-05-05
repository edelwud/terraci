package generate

import (
	"github.com/edelwud/terraci/pkg/execution"
	"github.com/edelwud/terraci/pkg/pipeline"
	configpkg "github.com/edelwud/terraci/plugins/github/internal/config"
	domainpkg "github.com/edelwud/terraci/plugins/github/internal/domain"
)

// Generator transforms a pipeline IR into GitHub Actions workflow YAML.
// The IR is bound at construction time.
type Generator struct {
	settings settings
	ir       *pipeline.IR
}

// NewGenerator creates a new GitHub Actions pipeline generator bound to the
// supplied IR.
func NewGenerator(cfg *configpkg.Config, execCfg execution.Config, ir *pipeline.IR) *Generator {
	return &Generator{
		settings: newSettings(cfg, execCfg),
		ir:       ir,
	}
}

func (g *Generator) Generate() (pipeline.GeneratedPipeline, error) {
	if g.ir == nil {
		return &domainpkg.Workflow{Jobs: map[string]*domainpkg.Job{}}, nil
	}
	return g.transform(g.ir), nil
}

func (g *Generator) DryRun() (*pipeline.DryRunResult, error) {
	if g.ir == nil {
		return &pipeline.DryRunResult{}, nil
	}
	return g.ir.DryRun(g.ir.ModuleCount()), nil
}

func (g *Generator) IsPREnabled() bool {
	return g.settings.prEnabled()
}

func (g *Generator) transform(ir *pipeline.IR) *domainpkg.Workflow {
	workflow := newWorkflowBuilder(g.settings).baseWorkflow()
	builder := newJobBuilder(g.settings)

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
