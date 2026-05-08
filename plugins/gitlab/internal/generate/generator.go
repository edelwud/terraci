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
		return &domain.Pipeline{Jobs: map[string]*domain.Job{}}, nil
	}
	return g.transform(g.ir)
}

// DryRun returns a summary of the bound IR without rendering YAML.
func (g *Generator) DryRun() (*pipeline.DryRunResult, error) {
	if g.ir == nil {
		return &pipeline.DryRunResult{}, nil
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

	result := &domain.Pipeline{
		Stages:    stagePlan.stages,
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

	builder := newJobBuilder(g.settings, stagePlan.stageByJob, func(job *domain.Job, jobType configpkg.JobOverwriteType) error {
		return applyResolvedJobConfig(g.settings, job, jobType)
	})
	for _, ref := range ir.JobRefs() {
		job, err := builder.renderJob(ref.Job)
		if err != nil {
			return nil, err
		}
		result.Jobs[ref.Job.Name] = job
	}

	return result, nil
}
