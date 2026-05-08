package generate

import (
	"fmt"
	"strings"

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

func (b jobBuilder) renderJob(irJob *pipeline.Job) (*domainpkg.Job, error) {
	profile, err := b.settings.jobProfile(jobOverwriteType(irJob))
	if err != nil {
		return nil, err
	}

	steps := []domainpkg.Step{checkoutStep()}
	for _, input := range irJob.InputArtifacts {
		if !input.Configured() {
			continue
		}
		steps = append(steps, downloadArtifactStep("Download "+input.Artifact.Name, input.Artifact.Name, input.Optional))
	}
	steps = append(steps, profile.stepsBefore...)
	scriptLines := cishell.RenderOperation(irJob.Operation)
	if irJob.AllowFailure {
		scriptLines = nil
		for _, command := range cishell.RenderOperation(irJob.Operation) {
			scriptLines = append(scriptLines, command+" || true")
		}
	}

	steps = append(steps, runStep(runStepName(irJob), strings.Join(scriptLines, "\n")))
	steps = append(steps, profile.stepsAfter...)
	if irJob.OutputArtifact.Configured() {
		var stageName, uploadName string
		switch irJob.Operation.Type {
		case pipeline.OperationTypeTerraformPlan:
			stageName = "Stage plan artifacts"
			uploadName = "Upload plan artifacts"
		case pipeline.OperationTypeCommands:
			stageName = fmt.Sprintf("Stage %s results", irJob.Name)
			uploadName = fmt.Sprintf("Upload %s results", irJob.Name)
		case pipeline.OperationTypeTerraformApply:
			stageName = fmt.Sprintf("Stage %s artifacts", irJob.Name)
			uploadName = fmt.Sprintf("Upload %s artifacts", irJob.Name)
		}
		steps = append(steps,
			stageArtifactStep(stageName, irJob.OutputArtifact, artifactRequired(irJob)),
			uploadArtifactStep(uploadName, irJob.OutputArtifact),
		)
	}

	job := &domainpkg.Job{
		RunsOn:      profile.runsOn,
		Needs:       pipeline.DependencyNames(irJob.Dependencies),
		Env:         mergeJobEnv(irJob.Env, profile.env),
		Steps:       steps,
		If:          profile.ifExpr,
		Environment: profile.environment,
	}
	if profile.container != nil {
		job.Container = profile.container
	}
	if irJob.Module != nil {
		job.Concurrency = &domainpkg.Concurrency{
			Group:            irJob.Module.ID(),
			CancelInProgress: false,
		}
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

func runStepName(irJob *pipeline.Job) string {
	if irJob == nil {
		return "Run"
	}
	if irJob.Module == nil {
		return "Run " + irJob.Name
	}
	switch irJob.Operation.Type {
	case pipeline.OperationTypeTerraformPlan:
		return "Plan " + irJob.Module.ID()
	case pipeline.OperationTypeTerraformApply:
		return "Apply " + irJob.Module.ID()
	case pipeline.OperationTypeCommands:
		return "Run " + irJob.Name
	default:
		return "Run"
	}
}

func artifactRequired(irJob *pipeline.Job) bool {
	return irJob != nil && irJob.Operation.Type == pipeline.OperationTypeTerraformPlan
}

func checkoutStep() domainpkg.Step {
	return domainpkg.Step{Name: "Checkout", Uses: "actions/checkout@v4"}
}

func runStep(name, script string) domainpkg.Step {
	return domainpkg.Step{Name: name, Run: script}
}

func downloadArtifactStep(name, artifact string, optional bool) domainpkg.Step {
	step := domainpkg.Step{
		Name: name,
		Uses: "actions/download-artifact@v4",
		With: map[string]string{
			"name": artifact,
			"path": ".",
		},
	}
	if optional {
		step.If = "always()"
		step.ContinueOnError = true
	}
	return step
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
