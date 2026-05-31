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
		return domainpkg.EmptyWorkflow(), nil
	}
	if err := g.ir.Validate(); err != nil {
		return nil, err
	}
	return g.transform(g.ir)
}

func (g *Generator) DryRun() (*pipeline.DryRunResult, error) {
	if g.ir == nil {
		return &pipeline.DryRunResult{}, nil
	}
	if err := g.ir.Validate(); err != nil {
		return nil, err
	}
	return g.ir.DryRun(g.ir.ModuleCount()), nil
}

func (g *Generator) transform(ir *pipeline.IR) (*domainpkg.Workflow, error) {
	workflow := domainpkg.NewWorkflowBuilder(domainpkg.WorkflowOptions{
		Name: "Terraform",
		On: domainpkg.WorkflowTrigger{
			Push:        &domainpkg.PushTrigger{Branches: []string{"main"}},
			PullRequest: &domainpkg.PRTrigger{Branches: []string{"main"}},
		},
		Permissions: g.settings.permissions(),
		Env:         g.settings.env(),
	})
	builder := newJobBuilder(g.settings)

	jobs := ir.Jobs()
	for i := range jobs {
		irJob := jobs[i]
		job, err := builder.renderJob(irJob)
		if err != nil {
			return nil, err
		}
		if err := workflow.AddJob(irJob.Name(), job); err != nil {
			return nil, err
		}
	}

	return workflow.Build()
}
