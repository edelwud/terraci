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

func (b jobBuilder) renderJob(irJob pipeline.Job) (domainpkg.Job, error) {
	profile, err := b.settings.jobProfile(jobOverwriteType(irJob))
	if err != nil {
		var zero domainpkg.Job
		return zero, err
	}

	steps := []domainpkg.Step{checkoutStep()}
	for _, input := range irJob.InputArtifacts() {
		if !input.Configured() {
			continue
		}
		steps = append(steps, downloadArtifactStep("Download "+input.Artifact.Name, input.Artifact.Name, input.Optional))
	}
	steps = append(steps, profile.stepsBefore...)
	operation := irJob.Operation()
	scriptLines := cishell.RenderOperation(operation)
	if irJob.AllowFailure() {
		scriptLines = nil
		for _, command := range cishell.RenderOperation(operation) {
			scriptLines = append(scriptLines, command+" || true")
		}
	}

	steps = append(steps, runStep(runStepName(irJob), strings.Join(scriptLines, "\n")))
	steps = append(steps, profile.stepsAfter...)
	outputArtifact := irJob.OutputArtifact()
	if outputArtifact.Configured() {
		var stageName, uploadName string
		switch operation.Type() {
		case pipeline.OperationTypeTerraformPlan:
			stageName = "Stage plan artifacts"
			uploadName = "Upload plan artifacts"
		case pipeline.OperationTypeCommands:
			stageName = fmt.Sprintf("Stage %s results", irJob.Name())
			uploadName = fmt.Sprintf("Upload %s results", irJob.Name())
		case pipeline.OperationTypeTerraformApply:
			stageName = fmt.Sprintf("Stage %s artifacts", irJob.Name())
			uploadName = fmt.Sprintf("Upload %s artifacts", irJob.Name())
		}
		steps = append(steps,
			stageArtifactStep(stageName, outputArtifact, artifactRequired(irJob)),
			uploadArtifactStep(uploadName, outputArtifact),
		)
	}

	job := domainpkg.JobOptions{
		RunsOn:      profile.runsOn,
		Needs:       pipeline.DependencyNames(irJob.Dependencies()),
		Env:         mergeJobEnv(irJob.Env(), profile.env),
		Steps:       steps,
		If:          profile.ifExpr,
		Environment: profile.environment,
	}
	if profile.container != nil {
		job.Container = profile.container
	}
	if module := irJob.Module(); module != nil {
		job.Concurrency = &domainpkg.Concurrency{
			Group:            module.ID(),
			CancelInProgress: false,
		}
	}
	return domainpkg.NewJob(job)
}

func jobOverwriteType(irJob pipeline.Job) configpkg.JobOverwriteType {
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

func runStepName(irJob pipeline.Job) string {
	module := irJob.Module()
	if module == nil {
		return "Run " + irJob.Name()
	}
	switch irJob.Operation().Type() {
	case pipeline.OperationTypeTerraformPlan:
		return "Plan " + module.ID()
	case pipeline.OperationTypeTerraformApply:
		return "Apply " + module.ID()
	case pipeline.OperationTypeCommands:
		return "Run " + irJob.Name()
	default:
		return "Run"
	}
}

func artifactRequired(irJob pipeline.Job) bool {
	return irJob.Operation().Type() == pipeline.OperationTypeTerraformPlan
}

func checkoutStep() domainpkg.Step {
	return domainpkg.NewStep(domainpkg.StepOptions{Name: "Checkout", Uses: "actions/checkout@v4"})
}

func runStep(name, script string) domainpkg.Step {
	return domainpkg.NewStep(domainpkg.StepOptions{Name: name, Run: script})
}

func downloadArtifactStep(name, artifact string, optional bool) domainpkg.Step {
	return domainpkg.NewStep(domainpkg.StepOptions{
		Name: name,
		Uses: "actions/download-artifact@v4",
		With: map[string]string{
			"name": artifact,
			"path": ".",
		},
		If:              optionalIf(optional),
		ContinueOnError: optional,
	})
}

func uploadArtifactStep(name string, artifact pipeline.Artifact) domainpkg.Step {
	return domainpkg.NewStep(domainpkg.StepOptions{
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
	})
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

	return domainpkg.NewStep(domainpkg.StepOptions{
		Name: name,
		Run:  strings.Join(lines, "\n"),
		If:   "always()",
	})
}

func optionalIf(optional bool) string {
	if optional {
		return "always()"
	}
	return ""
}

func artifactStageDir(artifact pipeline.Artifact) string {
	return ".terraci/artifacts/" + artifact.Name + "/"
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}
