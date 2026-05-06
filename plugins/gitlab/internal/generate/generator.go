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
	settings          settings
	stagePlanner      stagePlanner
	jobBuilder        jobBuilder
	contributionIndex contributionIndex
	ir                *pipeline.IR
}

// NewGenerator creates a new GitLab pipeline generator bound to the supplied IR.
func NewGenerator(cfg *configpkg.Config, execCfg execution.Config, ir *pipeline.IR) *Generator {
	cfgSettings := newSettings(cfg, execCfg)
	index := newContributionIndexFromIR(ir)
	return &Generator{
		settings:     cfgSettings,
		stagePlanner: newStagePlanner(cfgSettings, index),
		jobBuilder: newJobBuilder(cfgSettings, index, func(job *domain.Job, jobType configpkg.JobOverwriteType) error {
			return applyResolvedJobConfig(cfgSettings, job, jobType)
		}),
		contributionIndex: index,
		ir:                ir,
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
	result := g.ir.DryRun(g.ir.ModuleCount())
	result.Stages = len(g.stagePlanner.stages(g.ir))
	return result, nil
}

func (g *Generator) transform(ir *pipeline.IR) (*domain.Pipeline, error) {
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
				job, err := g.jobBuilder.planJob(mj.Plan, mj.Module, level.Index, prefix)
				if err != nil {
					return nil, err
				}
				result.Jobs[mj.Plan.Name] = job
			}
			if mj.Apply != nil {
				job, err := g.jobBuilder.applyJob(mj.Apply, mj.Module, level.Index, prefix)
				if err != nil {
					return nil, err
				}
				result.Jobs[mj.Apply.Name] = job
			}
		}
	}

	if g.contributionIndex.hasContributedJobs() {
		for i := range ir.Jobs {
			cj := &ir.Jobs[i]
			job, err := g.jobBuilder.contributedJob(cj)
			if err != nil {
				return nil, err
			}
			if cj.Phase == pipeline.PhaseFinalize {
				g.jobBuilder.applySummaryOverrides(job)
			}
			result.Jobs[cj.Name] = job
		}
	}

	return result, nil
}

// IsMREnabled returns true if MR integration is enabled in config.
func (g *Generator) IsMREnabled() bool {
	return g.settings.mrCommentEnabled()
}

func (g *Generator) stagesPrefix() string {
	return g.settings.stagesPrefix()
}
