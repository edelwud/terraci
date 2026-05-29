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
	applyConfig func(job *domain.JobOptions, jobType configpkg.JobOverwriteType) error
}

func newJobBuilder(settings settings, stageByJob map[string]string, applyConfig func(job *domain.JobOptions, jobType configpkg.JobOverwriteType) error) jobBuilder {
	return jobBuilder{
		settings:    settings,
		stageByJob:  stageByJob,
		applyConfig: applyConfig,
	}
}

func (b jobBuilder) renderJob(irJob *pipeline.Job) (domain.Job, error) {
	script := cishell.RenderOperation(irJob.Operation())
	if irJob.AllowFailure() {
		script = allowFailureScript(script)
	}

	job := domain.JobOptions{
		Stage:        b.stageByJob[irJob.Name()],
		Script:       script,
		Variables:    copyStringMap(irJob.Env()),
		Artifacts:    defaultArtifacts(irJob.OutputArtifact()),
		Needs:        jobNeeds(irJob.Dependencies(), irJob.InputArtifacts()),
		AllowFailure: irJob.AllowFailure(),
	}

	if module := irJob.Module(); module != nil {
		job.Cache = b.cache(module)
		job.ResourceGroup = module.ID()
	}

	if err := b.applyConfig(&job, jobOverwriteType(irJob)); err != nil {
		var zero domain.Job
		return zero, err
	}
	return domain.NewJob(job)
}

func jobOverwriteType(irJob *pipeline.Job) configpkg.JobOverwriteType {
	if irJob == nil {
		return ""
	}
	switch irJob.Operation().Type() {
	case pipeline.OperationTypeTerraformPlan:
		return configpkg.OverwriteTypePlan
	case pipeline.OperationTypeTerraformApply:
		return configpkg.OverwriteTypeApply
	case pipeline.OperationTypeCommands:
		return configpkg.JobOverwriteType(irJob.Name())
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
	return strings.ReplaceAll(module.ID(), "/", "-")
}

func cachePaths(module *discovery.Module, templates []string) []string {
	if len(templates) == 0 {
		return []string{module.ID() + "/.terraform/"}
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
		return []string{module.ID() + "/.terraform/"}
	}

	return paths
}

func renderCacheTemplate(template string, module *discovery.Module, fallback string) string {
	if strings.TrimSpace(template) == "" {
		return fallback
	}

	replacer := strings.NewReplacer(
		"{module_path}", module.ID(),
		"{service}", module.Get("service"),
		"{environment}", module.Get("environment"),
		"{region}", module.Get("region"),
		"{module}", module.Get("module"),
	)

	return replacer.Replace(template)
}

func jobNeeds(deps []pipeline.JobDependency, inputs []pipeline.InputArtifact) []domain.JobNeed {
	if len(deps) == 0 && len(inputs) == 0 {
		return nil
	}

	artifactInputs := make(map[string]pipeline.InputArtifact, len(inputs))
	for _, input := range inputs {
		if !input.Configured() {
			continue
		}
		artifactInputs[input.ProducerJob] = input
	}

	needs := make([]domain.JobNeed, 0, len(deps)+len(artifactInputs))
	seen := make(map[string]struct{}, len(deps)+len(artifactInputs))
	for _, dep := range deps {
		if dep.Job == "" {
			continue
		}
		need := domain.JobNeed{Job: dep.Job}
		if input, ok := artifactInputs[dep.Job]; ok {
			need.Artifacts = artifactNeedPtr(true)
			need.Optional = input.Optional
		} else {
			need.Artifacts = artifactNeedPtr(false)
		}
		needs = append(needs, need)
		seen[dep.Job] = struct{}{}
	}
	for producerJob, input := range artifactInputs {
		if _, ok := seen[producerJob]; ok {
			continue
		}
		needs = append(needs, domain.JobNeed{
			Job:       producerJob,
			Optional:  input.Optional,
			Artifacts: artifactNeedPtr(true),
		})
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
