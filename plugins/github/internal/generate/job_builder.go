package generate

import (
	"fmt"
	"strings"

	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/pipeline"
	configpkg "github.com/edelwud/terraci/plugins/github/internal/config"
	domainpkg "github.com/edelwud/terraci/plugins/github/internal/domain"
)

const stepsInitialCap = 8

type jobBuilder struct {
	settings settings
}

func newJobBuilder(settings settings) jobBuilder {
	return jobBuilder{settings: settings}
}

func (b jobBuilder) planJob(irJob *pipeline.Job, module *discovery.Module) *domainpkg.Job {
	runScript := strings.Join(pipeline.RenderOperationScript(irJob.Operation), "\n")
	steps := make([]domainpkg.Step, 0, stepsInitialCap)
	steps = append(steps, checkoutStep())
	steps = append(steps, b.settings.stepsBefore(configpkg.OverwriteTypePlan)...)
	steps = append(steps, phaseSteps(irJob.Steps, pipeline.PhasePrePlan)...)
	steps = append(steps, runStep("Plan "+module.ID(), runScript))
	steps = append(steps, phaseSteps(irJob.Steps, pipeline.PhasePostPlan)...)
	steps = append(steps, b.settings.stepsAfter(configpkg.OverwriteTypePlan)...)
	steps = append(steps, uploadArtifactStep("Upload plan artifacts", irJob.Name, irJob.ArtifactPaths))

	job := &domainpkg.Job{
		RunsOn: b.settings.runsOn(),
		Env:    irJob.Env,
		Concurrency: &domainpkg.Concurrency{
			Group:            module.ID(),
			CancelInProgress: false,
		},
		Steps: steps,
	}
	if container := b.settings.container(); container != nil {
		job.Container = container
	}

	job.Needs = irJob.Dependencies
	return job
}

func (b jobBuilder) applyJob(irJob *pipeline.Job, module *discovery.Module) *domainpkg.Job {
	runScript := strings.Join(pipeline.RenderOperationScript(irJob.Operation), "\n")
	steps := []domainpkg.Step{checkoutStep()}
	if b.settings.planEnabled() {
		steps = append(steps, downloadArtifactStep("Download plan artifacts", pipeline.JobName("plan", module)))
	}
	steps = append(steps, b.settings.stepsBefore(configpkg.OverwriteTypeApply)...)
	steps = append(steps, phaseSteps(irJob.Steps, pipeline.PhasePreApply)...)
	steps = append(steps, runStep("Apply "+module.ID(), runScript))
	steps = append(steps, phaseSteps(irJob.Steps, pipeline.PhasePostApply)...)
	steps = append(steps, b.settings.stepsAfter(configpkg.OverwriteTypeApply)...)

	job := &domainpkg.Job{
		RunsOn: b.settings.runsOn(),
		Env:    irJob.Env,
		Concurrency: &domainpkg.Concurrency{
			Group:            module.ID(),
			CancelInProgress: false,
		},
		Steps: steps,
		Needs: irJob.Dependencies,
	}
	if container := b.settings.container(); container != nil {
		job.Container = container
	}
	if !b.settings.autoApprove() {
		job.Environment = "production"
	}
	return job
}

func (b jobBuilder) contributedJob(irJob *pipeline.Job) *domainpkg.Job {
	scriptLines := pipeline.RenderOperationScript(irJob.Operation)
	if irJob.AllowFailure {
		scriptLines = nil
		for _, command := range pipeline.RenderOperationScript(irJob.Operation) {
			scriptLines = append(scriptLines, command+" || true")
		}
	}

	steps := []domainpkg.Step{
		checkoutStep(),
		downloadAllArtifactsStep(),
	}
	steps = append(steps, b.settings.defaultStepsBefore()...)
	steps = append(steps, b.settings.overwriteStepsBefore(irJob.Name)...)
	steps = append(steps, runStep("Run "+irJob.Name, strings.Join(scriptLines, "\n")))
	steps = append(steps, b.settings.defaultStepsAfter()...)
	steps = append(steps, b.settings.overwriteStepsAfter(irJob.Name)...)
	if len(irJob.ArtifactPaths) > 0 {
		steps = append(steps, uploadArtifactStep(fmt.Sprintf("Upload %s results", irJob.Name), irJob.Name+"-results", irJob.ArtifactPaths))
	}

	runsOn := b.settings.runsOn()
	if ow := b.settings.overwriteRunsOn(irJob.Name); ow != "" {
		runsOn = ow
	}

	job := &domainpkg.Job{
		RunsOn: runsOn,
		Needs:  irJob.Dependencies,
		Steps:  steps,
	}
	if container := b.settings.overwriteContainer(irJob.Name); container != nil {
		job.Container = container
	} else if container := b.settings.container(); container != nil {
		job.Container = container
	}
	if env := b.settings.overwriteEnv(irJob.Name); len(env) > 0 {
		job.Env = env
	}
	if summaryRunsOn := b.settings.summaryRunsOn(); irJob.Phase == pipeline.PhaseFinalize && summaryRunsOn != "" {
		job.RunsOn = summaryRunsOn
	}
	if irJob.Phase == pipeline.PhaseFinalize {
		job.If = "github.event_name == 'pull_request'"
	}
	return job
}

func checkoutStep() domainpkg.Step {
	return domainpkg.Step{Name: "Checkout", Uses: "actions/checkout@v4"}
}

func runStep(name, script string) domainpkg.Step {
	return domainpkg.Step{Name: name, Run: script}
}

func downloadArtifactStep(name, artifact string) domainpkg.Step {
	return domainpkg.Step{
		Name: name,
		Uses: "actions/download-artifact@v4",
		With: map[string]string{
			"name": artifact,
		},
	}
}

func downloadAllArtifactsStep() domainpkg.Step {
	return domainpkg.Step{
		Name: "Download all plan artifacts",
		Uses: "actions/download-artifact@v4",
	}
}

func uploadArtifactStep(name, artifact string, paths []string) domainpkg.Step {
	return domainpkg.Step{
		Name: name,
		Uses: "actions/upload-artifact@v4",
		With: map[string]string{
			"name":           artifact,
			"path":           strings.Join(paths, "\n"),
			"retention-days": "1",
		},
		If: "always()",
	}
}

func phaseSteps(steps []pipeline.Step, phase pipeline.Phase) []domainpkg.Step {
	var result []domainpkg.Step
	for _, step := range steps {
		if step.Phase != phase {
			continue
		}
		result = append(result, domainpkg.Step{Name: step.Name, Run: step.Command})
	}
	return result
}
