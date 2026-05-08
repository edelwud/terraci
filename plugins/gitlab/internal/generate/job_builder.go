package generate

import (
	"maps"
	"strings"

	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/pipeline"
	"github.com/edelwud/terraci/pkg/pipeline/cishell"
	configpkg "github.com/edelwud/terraci/plugins/gitlab/internal/config"
	"github.com/edelwud/terraci/plugins/gitlab/internal/domain"
)

type jobBuilder struct {
	settings    settings
	stageByJob  map[string]string
	applyConfig func(job *domain.Job, jobType configpkg.JobOverwriteType) error
}

func newJobBuilder(settings settings, stageByJob map[string]string, applyConfig func(job *domain.Job, jobType configpkg.JobOverwriteType) error) jobBuilder {
	return jobBuilder{
		settings:    settings,
		stageByJob:  stageByJob,
		applyConfig: applyConfig,
	}
}

func (b jobBuilder) renderJob(irJob *pipeline.Job) (*domain.Job, error) {
	script := cishell.RenderOperation(irJob.Operation)
	if irJob.AllowFailure {
		script = allowFailureScript(script)
	}

	job := &domain.Job{
		Stage:        b.stageByJob[irJob.Name],
		Script:       script,
		Variables:    copyStringMap(irJob.Env),
		Artifacts:    defaultArtifacts(irJob.OutputArtifact),
		Needs:        jobNeeds(irJob.Dependencies),
		AllowFailure: irJob.AllowFailure,
	}

	if irJob.Module != nil {
		job.Cache = b.cache(irJob.Module)
		job.ResourceGroup = irJob.Module.ID()
	}

	if err := b.applyConfig(job, jobOverwriteType(irJob)); err != nil {
		return nil, err
	}
	return job, nil
}

func jobOverwriteType(irJob *pipeline.Job) configpkg.JobOverwriteType {
	if irJob == nil {
		return ""
	}
	switch irJob.Operation.Type {
	case pipeline.OperationTypeTerraformPlan:
		return configpkg.OverwriteTypePlan
	case pipeline.OperationTypeTerraformApply:
		return configpkg.OverwriteTypeApply
	case pipeline.OperationTypeCommands:
		return configpkg.JobOverwriteType(irJob.Name)
	default:
		return ""
	}
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

func jobNeeds(deps []pipeline.JobDependency) []domain.JobNeed {
	if len(deps) == 0 {
		return nil
	}

	needs := make([]domain.JobNeed, len(deps))
	for i, dep := range deps {
		needs[i] = domain.JobNeed{
			Job:       dep.Job,
			Optional:  dep.Optional,
			Artifacts: artifactNeedPtr(dep.Artifacts),
		}
	}
	return needs
}

func defaultArtifacts(artifact pipeline.Artifact) *domain.Artifacts {
	if !artifact.Configured() {
		return nil
	}

	return &domain.Artifacts{
		Name:     artifact.Name,
		Paths:    artifact.Paths,
		ExpireIn: "1 day",
		When:     "always",
	}
}

func artifactNeedPtr(value bool) *bool {
	return &value
}

func allowFailureScript(script []string) []string {
	result := make([]string, 0, len(script))
	for _, cmd := range script {
		result = append(result, cmd+" || true")
	}
	return result
}

func copyStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	return maps.Clone(in)
}
