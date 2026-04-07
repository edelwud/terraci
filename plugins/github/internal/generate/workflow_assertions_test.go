package generate

import (
	"strings"
	"testing"

	"github.com/edelwud/terraci/pkg/ci/citest"
	domainpkg "github.com/edelwud/terraci/plugins/github/internal/domain"
)

type workflowAssert struct {
	t        *testing.T
	workflow *domainpkg.Workflow
}

func assertWorkflow(t *testing.T, workflow *domainpkg.Workflow) *workflowAssert {
	t.Helper()
	return &workflowAssert{t: t, workflow: workflow}
}

func (a *workflowAssert) jobCount(expected int) *workflowAssert {
	a.t.Helper()
	if got := len(a.workflow.Jobs); got != expected {
		a.t.Fatalf("expected %d jobs, got %d", expected, got)
	}
	return a
}

func (a *workflowAssert) hasJob(name string) *workflowAssert {
	a.t.Helper()
	if _, ok := a.workflow.Jobs[name]; !ok {
		a.t.Fatalf("expected job %q to exist", name)
	}
	return a
}

func (a *workflowAssert) noJob(name string) *workflowAssert {
	a.t.Helper()
	if _, ok := a.workflow.Jobs[name]; ok {
		a.t.Fatalf("expected job %q to not exist", name)
	}
	return a
}

func (a *workflowAssert) env(name, expected string) *workflowAssert {
	a.t.Helper()
	if got := a.workflow.Env[name]; got != expected {
		a.t.Fatalf("expected env %s=%q, got %q", name, expected, got)
	}
	return a
}

func (a *workflowAssert) job(name string) *jobAssert {
	a.t.Helper()
	job := a.workflow.Jobs[name]
	if job == nil {
		a.t.Fatalf("expected job %q to exist", name)
	}
	return &jobAssert{t: a.t, name: name, job: job}
}

type jobAssert struct {
	t    *testing.T
	name string
	job  *domainpkg.Job
}

func (a *jobAssert) hasNeed(name string) *jobAssert {
	a.t.Helper()
	citest.AssertHasNeed(a.t, a.name, a.job.Needs, name)
	return a
}

func (a *jobAssert) noNeedWithPrefix(prefix string) *jobAssert {
	a.t.Helper()
	citest.AssertNoNeedWithPrefix(a.t, a.name, a.job.Needs, prefix)
	return a
}

func (a *jobAssert) noEnvironment() *jobAssert {
	a.t.Helper()
	if a.job.Environment != "" {
		a.t.Fatalf("expected job %q to have no environment, got %q", a.name, a.job.Environment)
	}
	return a
}

func (a *jobAssert) environment(expected string) *jobAssert {
	a.t.Helper()
	if a.job.Environment != expected {
		a.t.Fatalf("expected job %q environment=%q, got %q", a.name, expected, a.job.Environment)
	}
	return a
}

func (a *jobAssert) containerImage(expected string) *jobAssert {
	a.t.Helper()
	if a.job.Container == nil {
		a.t.Fatalf("expected job %q to have container", a.name)
	}
	if a.job.Container.Image != expected {
		a.t.Fatalf("expected job %q container=%q, got %q", a.name, expected, a.job.Container.Image)
	}
	return a
}

func (a *jobAssert) stepUses(action string) *jobAssert {
	a.t.Helper()
	for _, step := range a.job.Steps {
		if step.Uses == action {
			return a
		}
	}
	a.t.Fatalf("expected job %q to use %q", a.name, action)
	return a
}

func (a *jobAssert) stepRunContains(fragment string) *jobAssert {
	a.t.Helper()
	for _, step := range a.job.Steps {
		if strings.Contains(step.Run, fragment) {
			return a
		}
	}
	a.t.Fatalf("expected job %q run steps to contain %q", a.name, fragment)
	return a
}

func (a *jobAssert) stepNamed(name string) *jobAssert {
	a.t.Helper()
	for _, step := range a.job.Steps {
		if step.Name == name {
			return a
		}
	}
	a.t.Fatalf("expected job %q to have step named %q", a.name, name)
	return a
}
