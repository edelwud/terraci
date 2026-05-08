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
	return g.transform(g.ir)
}

func (g *Generator) DryRun() (*pipeline.DryRunResult, error) {
	if g.ir == nil {
		return &pipeline.DryRunResult{}, nil
	}
	return g.ir.DryRun(g.ir.ModuleCount()), nil
}

func (g *Generator) transform(ir *pipeline.IR) (*domainpkg.Workflow, error) {
	workflow := newWorkflowBuilder(g.settings).baseWorkflow()
	builder := newJobBuilder(g.settings)

	for _, ref := range ir.JobRefs() {
		job, err := builder.renderJob(ref.Job)
		if err != nil {
			return nil, err
		}
		workflow.Jobs[ref.Job.Name] = job
	}

	return workflow, nil
}
