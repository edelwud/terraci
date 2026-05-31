package generate

import (
	"github.com/edelwud/terraci/pkg/execution"
	"github.com/edelwud/terraci/pkg/pipeline"
	configpkg "github.com/edelwud/terraci/plugins/gitlab/internal/config"
	"github.com/edelwud/terraci/plugins/gitlab/internal/domain"
)

const (
	DefaultStagesPrefix = "deploy"
	WhenManual          = "manual"
)

// Generator transforms TerraCi IR into GitLab CI domain models. The IR is
// bound at construction time — the IR carries every module + contribution
// the pipeline should render.
type Generator struct {
	settings     settings
	stagePlanner stagePlanner
	ir           *pipeline.IR
}

// NewGenerator creates a new GitLab pipeline generator bound to the supplied IR.
func NewGenerator(cfg *configpkg.Config, execCfg execution.Config, ir *pipeline.IR) *Generator {
	cfgSettings := newSettings(cfg, execCfg)
	return &Generator{
		settings:     cfgSettings,
		stagePlanner: newStagePlanner(cfgSettings),
		ir:           ir,
	}
}

// Generate creates a GitLab CI pipeline from the bound IR.
func (g *Generator) Generate() (pipeline.GeneratedPipeline, error) {
	if g.ir == nil {
		return domain.EmptyPipeline(), nil
	}
	if err := g.ir.Validate(); err != nil {
		return nil, err
	}
	return g.transform(g.ir)
}

// DryRun returns a summary of the bound IR without rendering YAML.
func (g *Generator) DryRun() (*pipeline.DryRunResult, error) {
	if g.ir == nil {
		return &pipeline.DryRunResult{}, nil
	}
	if err := g.ir.Validate(); err != nil {
		return nil, err
	}
	plan, err := g.stagePlanner.plan(g.ir)
	if err != nil {
		return nil, err
	}
	result := g.ir.DryRun(g.ir.ModuleCount())
	result.Stages = len(plan.stages)
	return result, nil
}

func (g *Generator) transform(ir *pipeline.IR) (*domain.Pipeline, error) {
	effectiveImage := g.settings.defaultImage()
	stagePlan, err := g.stagePlanner.plan(ir)
	if err != nil {
		return nil, err
	}

	builder := domain.NewPipelineBuilder(domain.PipelineOptions{
		Stages:    stagePlan.stages,
		Variables: g.settings.variables(),
		Default: &domain.DefaultConfig{
			Image: &domain.ImageConfig{
				Name:       effectiveImage.Name,
				Entrypoint: effectiveImage.Entrypoint,
			},
		},
		Workflow: g.generateWorkflow(),
	})

	jobBuilder := newJobBuilder(g.settings, stagePlan.stageByJob)
	jobs := ir.Jobs()
	for i := range jobs {
		irJob := jobs[i]
		job, err := jobBuilder.renderJob(irJob)
		if err != nil {
			return nil, err
		}
		if err := builder.AddJob(irJob.Name(), job); err != nil {
			return nil, err
		}
	}

	return builder.Build()
}
