package generate

import (
	"fmt"
	"strings"

	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/pipeline"
	"github.com/edelwud/terraci/pkg/pipeline/cishell"
	configpkg "github.com/edelwud/terraci/plugins/gitlab/internal/config"
	"github.com/edelwud/terraci/plugins/gitlab/internal/domain"
)

type jobBuilder struct {
	settings      settings
	contributions contributionIndex
	applyConfig   func(job *domain.Job, jobType configpkg.JobOverwriteType)
}

func newJobBuilder(settings settings, contributions contributionIndex, applyConfig func(job *domain.Job, jobType configpkg.JobOverwriteType)) jobBuilder {
	return jobBuilder{
		settings:      settings,
		contributions: contributions,
		applyConfig:   applyConfig,
	}
}

func (b jobBuilder) planJob(irJob *pipeline.Job, module *discovery.Module, levelIdx int, prefix string) *domain.Job {
	job := &domain.Job{
		Stage:         fmt.Sprintf("%s-plan-%d", prefix, levelIdx),
		Script:        b.scriptWithSteps(cishell.RenderOperation(irJob.Operation), irJob.Steps, pipeline.PhasePrePlan, pipeline.PhasePostPlan),
		Variables:     irJob.Env,
		Artifacts:     defaultArtifacts(irJob.ArtifactPaths),
		Cache:         b.cache(module),
		ResourceGroup: module.ID(),
		Needs:         requiredNeeds(irJob.Dependencies),
	}

	b.applyConfig(job, configpkg.OverwriteTypePlan)
	return job
}

func (b jobBuilder) applyJob(irJob *pipeline.Job, module *discovery.Module, levelIdx int, prefix string) *domain.Job {
	job := &domain.Job{
		Stage:         fmt.Sprintf("%s-apply-%d", prefix, levelIdx),
		Script:        b.scriptWithSteps(cishell.RenderOperation(irJob.Operation), irJob.Steps, pipeline.PhasePreApply, pipeline.PhasePostApply),
		Variables:     irJob.Env,
		Cache:         b.cache(module),
		ResourceGroup: module.ID(),
		Needs:         requiredNeeds(irJob.Dependencies),
	}

	if !b.settings.autoApprove() {
		job.When = WhenManual
	}

	b.applyConfig(job, configpkg.OverwriteTypeApply)
	return job
}

func (b jobBuilder) contributedJob(irJob *pipeline.Job) *domain.Job {
	job := &domain.Job{
		Stage:  b.contributions.stageFor(irJob.Name),
		Script: contributedScript(cishell.RenderOperation(irJob.Operation), irJob.AllowFailure),
		Needs:  optionalNeeds(irJob.Dependencies),
	}

	if artifacts := defaultArtifacts(irJob.ArtifactPaths); artifacts != nil {
		job.Artifacts = artifacts
	}

	b.applyConfig(job, configpkg.JobOverwriteType(irJob.Name))

	return job
}

func (b jobBuilder) applySummaryOverrides(job *domain.Job) {
	job.Rules = []domain.Rule{{If: "$CI_MERGE_REQUEST_IID", When: "always"}}

	summary := b.settings.summaryJob()
	if !summary.configured() {
		return
	}

	if summary.image != nil && summary.image.Name != "" {
		job.Image = &domain.ImageConfig{
			Name:       summary.image.Name,
			Entrypoint: summary.image.Entrypoint,
		}
	}
	if len(summary.tags) > 0 {
		job.Tags = summary.tags
	}
}

func (b jobBuilder) scriptWithSteps(coreScript []string, steps []pipeline.Step, prePh, postPh pipeline.Phase) []string {
	var script []string
	for _, step := range steps {
		if step.Phase == prePh {
			script = append(script, step.Command)
		}
	}
	script = append(script, coreScript...)
	for _, step := range steps {
		if step.Phase == postPh {
			script = append(script, step.Command)
		}
	}
	return script
}

func (b jobBuilder) cache(module *discovery.Module) *domain.Cache {
	if !b.settings.cacheEnabled() {
		return nil
	}

	return &domain.Cache{
		Key:    renderCacheTemplate(b.settings.cacheKeyTemplate(), module, cacheKey(module)),
		Paths:  cachePaths(module, b.settings.cachePathTemplates()),
		Policy: b.settings.cachePolicy(),
	}
}

func cacheKey(module *discovery.Module) string {
	return strings.ReplaceAll(module.RelativePath, "/", "-")
}

func cachePaths(module *discovery.Module, templates []string) []string {
	if len(templates) == 0 {
		return []string{module.RelativePath + "/.terraform/"}
	}

	paths := make([]string, 0, len(templates))
	for _, template := range templates {
		path := renderCacheTemplate(template, module, "")
		if path == "" {
			continue
		}
		paths = append(paths, path)
	}

	if len(paths) == 0 {
		return []string{module.RelativePath + "/.terraform/"}
	}

	return paths
}

func renderCacheTemplate(template string, module *discovery.Module, fallback string) string {
	if strings.TrimSpace(template) == "" {
		return fallback
	}

	replacer := strings.NewReplacer(
		"{module_path}", module.RelativePath,
		"{service}", module.Get("service"),
		"{environment}", module.Get("environment"),
		"{region}", module.Get("region"),
		"{module}", module.Get("module"),
	)

	return replacer.Replace(template)
}

func requiredNeeds(deps []string) []domain.JobNeed {
	if len(deps) == 0 {
		return nil
	}

	needs := make([]domain.JobNeed, len(deps))
	for i, name := range deps {
		needs[i] = domain.JobNeed{Job: name}
	}
	return needs
}

func optionalNeeds(deps []string) []domain.JobNeed {
	if len(deps) == 0 {
		return nil
	}

	needs := make([]domain.JobNeed, len(deps))
	for i, name := range deps {
		needs[i] = domain.JobNeed{Job: name, Optional: true}
	}
	return needs
}

func defaultArtifacts(paths []string) *domain.Artifacts {
	if len(paths) == 0 {
		return nil
	}

	return &domain.Artifacts{
		Paths:    paths,
		ExpireIn: "1 day",
		When:     "always",
	}
}

func contributedScript(script []string, allowFailure bool) []string {
	if !allowFailure {
		return script
	}

	result := make([]string, 0, len(script))
	for _, cmd := range script {
		result = append(result, cmd+" || true")
	}
	return result
}
