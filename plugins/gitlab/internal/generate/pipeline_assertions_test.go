package generate

import (
	"slices"
	"strings"
	"testing"

	"github.com/edelwud/terraci/pkg/ci/citest"
)

type pipelineAssert struct {
	t        *testing.T
	pipeline *Pipeline
}

func assertPipeline(t *testing.T, pipeline *Pipeline) *pipelineAssert {
	t.Helper()
	return &pipelineAssert{t: t, pipeline: pipeline}
}

func (a *pipelineAssert) hasJob(name string) *pipelineAssert {
	a.t.Helper()
	if _, ok := a.pipeline.Jobs[name]; !ok {
		a.t.Fatalf("expected job %q to exist", name)
	}
	return a
}

func (a *pipelineAssert) noJob(name string) *pipelineAssert {
	a.t.Helper()
	if _, ok := a.pipeline.Jobs[name]; ok {
		a.t.Fatalf("expected job %q to not exist", name)
	}
	return a
}

func (a *pipelineAssert) jobCount(expected int) *pipelineAssert {
	a.t.Helper()
	if got := len(a.pipeline.Jobs); got != expected {
		a.t.Fatalf("expected %d jobs, got %d", expected, got)
	}
	return a
}

func (a *pipelineAssert) stageCount(expected int) *pipelineAssert {
	a.t.Helper()
	if got := len(a.pipeline.Stages); got != expected {
		a.t.Fatalf("expected %d stages, got %d: %v", expected, got, a.pipeline.Stages)
	}
	return a
}

func (a *pipelineAssert) hasStage(name string) *pipelineAssert {
	a.t.Helper()
	if !slices.Contains(a.pipeline.Stages, name) {
		a.t.Fatalf("expected stage %q to exist in %v", name, a.pipeline.Stages)
	}
	return a
}

func (a *pipelineAssert) noStage(name string) *pipelineAssert {
	a.t.Helper()
	if slices.Contains(a.pipeline.Stages, name) {
		a.t.Fatalf("expected stage %q to not exist in %v", name, a.pipeline.Stages)
	}
	return a
}

func (a *pipelineAssert) stagesHavePrefix(prefix string) *pipelineAssert {
	a.t.Helper()
	for _, stage := range a.pipeline.Stages {
		if !strings.HasPrefix(stage, prefix) {
			a.t.Fatalf("expected stage %q to have prefix %q", stage, prefix)
		}
	}
	return a
}

//nolint:unparam // Fluent assertion API keeps a consistent shape across helpers.
func (a *pipelineAssert) noStageWithFragment(fragment string) *pipelineAssert {
	a.t.Helper()
	for _, stage := range a.pipeline.Stages {
		if strings.Contains(stage, fragment) {
			a.t.Fatalf("expected no stage containing %q, got %q", fragment, stage)
		}
	}
	return a
}

func (a *pipelineAssert) variable(name, expected string) *pipelineAssert {
	a.t.Helper()
	if got := a.pipeline.Variables[name]; got != expected {
		a.t.Fatalf("expected variable %s=%q, got %q", name, expected, got)
	}
	return a
}

func (a *pipelineAssert) stageIndex(stage string) int {
	a.t.Helper()
	for i, current := range a.pipeline.Stages {
		if current == stage {
			return i
		}
	}
	a.t.Fatalf("expected stage %q to exist in %v", stage, a.pipeline.Stages)
	return -1
}

func (a *pipelineAssert) jobStageBefore(earlierJob, laterJob string) *pipelineAssert {
	a.t.Helper()
	earlier := a.job(earlierJob)
	later := a.job(laterJob)
	if a.stageIndex(earlier.job.Stage) >= a.stageIndex(later.job.Stage) {
		a.t.Fatalf(
			"expected job %q stage %q to be before job %q stage %q",
			earlierJob,
			earlier.job.Stage,
			laterJob,
			later.job.Stage,
		)
	}
	return a
}

func (a *pipelineAssert) job(name string) *jobAssert {
	a.t.Helper()
	job := a.pipeline.Jobs[name]
	if job == nil {
		a.t.Fatalf("expected job %q to exist", name)
	}
	return &jobAssert{
		t:        a.t,
		name:     name,
		job:      job,
		pipeline: a.pipeline,
	}
}

type jobAssert struct {
	t        *testing.T
	name     string
	job      *Job
	pipeline *Pipeline
}

func (a *jobAssert) hasNeed(name string) *jobAssert {
	a.t.Helper()
	citest.AssertHasNeed(a.t, a.name, a.needNames(), name)
	return a
}

func (a *jobAssert) noNeed(name string) *jobAssert {
	a.t.Helper()
	citest.AssertNoNeed(a.t, a.name, a.needNames(), name)
	return a
}

//nolint:unparam // Fluent assertion API keeps a consistent shape across helpers.
func (a *jobAssert) noNeedWithPrefix(prefix string) *jobAssert {
	a.t.Helper()
	citest.AssertNoNeedWithPrefix(a.t, a.name, a.needNames(), prefix)
	return a
}

//nolint:unparam // Fluent assertion API keeps a consistent shape across helpers.
func (a *jobAssert) noNeeds() *jobAssert {
	a.t.Helper()
	if len(a.job.Needs) != 0 {
		a.t.Fatalf("expected job %q to have no needs, got %#v", a.name, a.job.Needs)
	}
	return a
}

func (a *jobAssert) when(expected string) *jobAssert {
	a.t.Helper()
	if a.job.When != expected {
		a.t.Fatalf("expected job %q when=%q, got %q", a.name, expected, a.job.When)
	}
	return a
}

func (a *jobAssert) notManual() *jobAssert {
	a.t.Helper()
	if a.job.When == WhenManual {
		a.t.Fatalf("expected job %q to not be manual", a.name)
	}
	return a
}

//nolint:unparam // Fluent assertion API keeps a consistent shape across helpers.
func (a *jobAssert) resourceGroup(expected string) *jobAssert {
	a.t.Helper()
	if a.job.ResourceGroup != expected {
		a.t.Fatalf("expected job %q resource_group=%q, got %q", a.name, expected, a.job.ResourceGroup)
	}
	return a
}

//nolint:unparam // Fluent assertion API keeps a consistent shape across helpers.
func (a *jobAssert) variable(name, expected string) *jobAssert {
	a.t.Helper()
	if got := a.job.Variables[name]; got != expected {
		a.t.Fatalf("expected job %q variable %s=%q, got %q", a.name, name, expected, got)
	}
	return a
}

func (a *jobAssert) artifactPathContains(fragment string) *jobAssert {
	a.t.Helper()
	if a.job.Artifacts == nil {
		a.t.Fatalf("expected job %q to have artifacts", a.name)
	}
	for _, path := range a.job.Artifacts.Paths {
		if strings.Contains(path, fragment) {
			return a
		}
	}
	a.t.Fatalf("expected job %q artifacts to contain %q, got %v", a.name, fragment, a.job.Artifacts.Paths)
	return a
}

func (a *jobAssert) needNames() []string {
	needs := make([]string, 0, len(a.job.Needs))
	for _, need := range a.job.Needs {
		needs = append(needs, need.Job)
	}
	return needs
}
