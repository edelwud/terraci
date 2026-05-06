package generate

import (
	"fmt"
	"strings"

	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/pipeline"
	"github.com/edelwud/terraci/pkg/pipeline/cishell"
	configpkg "github.com/edelwud/terraci/plugins/github/internal/config"
	domainpkg "github.com/edelwud/terraci/plugins/github/internal/domain"
)

const (
	stepsInitialCap = 8
	actionTrue      = "true"
)

type jobBuilder struct {
	settings settings
}

func newJobBuilder(settings settings) jobBuilder {
	return jobBuilder{settings: settings}
}

func (b jobBuilder) planJob(irJob *pipeline.Job, module *discovery.Module) (*domainpkg.Job, error) {
	profile, err := b.settings.jobProfile(configpkg.OverwriteTypePlan)
	if err != nil {
		return nil, err
	}

	runScript := strings.Join(cishell.RenderOperation(irJob.Operation), "\n")
	steps := make([]domainpkg.Step, 0, stepsInitialCap)
	steps = append(steps, checkoutStep())
	steps = append(steps, profile.stepsBefore...)
	steps = append(steps, phaseSteps(irJob.Steps, pipeline.PhasePrePlan)...)
	steps = append(steps, runStep("Plan "+module.ID(), runScript))
	steps = append(steps, phaseSteps(irJob.Steps, pipeline.PhasePostPlan)...)
	steps = append(steps, profile.stepsAfter...)
	if irJob.OutputArtifact.Configured() {
		steps = append(steps,
			stageArtifactStep("Stage plan artifacts", irJob.OutputArtifact, true),
			uploadArtifactStep("Upload plan artifacts", irJob.OutputArtifact),
		)
	}

	job := &domainpkg.Job{
		RunsOn: profile.runsOn,
		Env:    mergeJobEnv(irJob.Env, profile.env),
		Concurrency: &domainpkg.Concurrency{
			Group:            module.ID(),
			CancelInProgress: false,
		},
		Steps: steps,
	}
	if profile.container != nil {
		job.Container = profile.container
	}

	job.Needs = pipeline.DependencyNames(irJob.Dependencies)
	return job, nil
}

func (b jobBuilder) applyJob(irJob *pipeline.Job, module *discovery.Module) (*domainpkg.Job, error) {
	profile, err := b.settings.jobProfile(configpkg.OverwriteTypeApply)
	if err != nil {
		return nil, err
	}

	runScript := strings.Join(cishell.RenderOperation(irJob.Operation), "\n")
	steps := []domainpkg.Step{checkoutStep()}
	for _, artifact := range irJob.InputArtifacts {
		steps = append(steps, downloadArtifactStep("Download "+artifact.Name, artifact.Name))
	}
	steps = append(steps, profile.stepsBefore...)
	steps = append(steps, phaseSteps(irJob.Steps, pipeline.PhasePreApply)...)
	steps = append(steps, runStep("Apply "+module.ID(), runScript))
	steps = append(steps, phaseSteps(irJob.Steps, pipeline.PhasePostApply)...)
	steps = append(steps, profile.stepsAfter...)

	job := &domainpkg.Job{
		RunsOn: profile.runsOn,
		Env:    mergeJobEnv(irJob.Env, profile.env),
		Concurrency: &domainpkg.Concurrency{
			Group:            module.ID(),
			CancelInProgress: false,
		},
		Steps: steps,
		Needs: pipeline.DependencyNames(irJob.Dependencies),
	}
	if profile.container != nil {
		job.Container = profile.container
	}
	if !b.settings.autoApprove() {
		job.Environment = "production"
	}
	return job, nil
}

func (b jobBuilder) contributedJob(irJob *pipeline.Job) (*domainpkg.Job, error) {
	profile, err := b.settings.jobProfile(configpkg.JobOverwriteType(irJob.Name))
	if err != nil {
		return nil, err
	}

	scriptLines := cishell.RenderOperation(irJob.Operation)
	if irJob.AllowFailure {
		scriptLines = nil
		for _, command := range cishell.RenderOperation(irJob.Operation) {
			scriptLines = append(scriptLines, command+" || true")
		}
	}

	steps := []domainpkg.Step{checkoutStep()}
	for _, artifact := range irJob.InputArtifacts {
		steps = append(steps, downloadArtifactStep("Download "+artifact.Name, artifact.Name))
	}
	steps = append(steps, profile.stepsBefore...)
	steps = append(steps, runStep("Run "+irJob.Name, strings.Join(scriptLines, "\n")))
	steps = append(steps, profile.stepsAfter...)
	if irJob.OutputArtifact.Configured() {
		steps = append(steps,
			stageArtifactStep(fmt.Sprintf("Stage %s results", irJob.Name), irJob.OutputArtifact, false),
			uploadArtifactStep(fmt.Sprintf("Upload %s results", irJob.Name), irJob.OutputArtifact),
		)
	}

	job := &domainpkg.Job{
		RunsOn: profile.runsOn,
		Needs:  pipeline.DependencyNames(irJob.Dependencies),
		Env:    mergeJobEnv(irJob.Env, profile.env),
		Steps:  steps,
	}
	if profile.container != nil {
		job.Container = profile.container
	}
	if summaryRunsOn := b.settings.summaryRunsOn(); irJob.Phase == pipeline.PhaseFinalize && summaryRunsOn != "" {
		job.RunsOn = summaryRunsOn
	}
	if irJob.Phase == pipeline.PhaseFinalize {
		job.If = "github.event_name == 'pull_request'"
	}
	return job, nil
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
			"path": ".",
		},
	}
}

func uploadArtifactStep(name string, artifact pipeline.Artifact) domainpkg.Step {
	return domainpkg.Step{
		Name: name,
		Uses: "actions/upload-artifact@v4",
		With: map[string]string{
			"name":                 artifact.Name,
			"path":                 artifactStageDir(artifact),
			"retention-days":       "1",
			"include-hidden-files": actionTrue,
			"if-no-files-found":    "warn",
		},
		If: "always()",
	}
}

func stageArtifactStep(name string, artifact pipeline.Artifact, required bool) domainpkg.Step {
	stageDir := artifactStageDir(artifact)
	lines := []string{
		"set -eu",
		"stage_dir=" + shellQuote(stageDir),
		`rm -rf "$stage_dir"`,
		`mkdir -p "$stage_dir"`,
	}
	for _, path := range artifact.Paths {
		lines = append(lines,
			"artifact_path="+shellQuote(path),
			`if [ -e "$artifact_path" ]; then`,
			`  artifact_dest="$stage_dir/$artifact_path"`,
			`  mkdir -p "$(dirname "$artifact_dest")"`,
			`  if [ -d "$artifact_path" ]; then cp -R "$artifact_path" "$artifact_dest"; else cp "$artifact_path" "$artifact_dest"; fi`,
			`fi`,
		)
	}
	if required {
		lines = append(lines,
			`if ! find "$stage_dir" -type f | grep -q .; then`,
			"  echo "+shellQuote("No artifact files staged for "+artifact.Name),
			"  exit 1",
			`fi`,
		)
	}

	return domainpkg.Step{
		Name: name,
		Run:  strings.Join(lines, "\n"),
		If:   "always()",
	}
}

func artifactStageDir(artifact pipeline.Artifact) string {
	return ".terraci/artifacts/" + artifact.Name + "/"
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
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
