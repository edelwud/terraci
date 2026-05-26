package pipeline

import (
	"testing"

	"github.com/edelwud/terraci/pkg/discovery"
)

func TestIRJobQueriesReturnDefensiveCopies(t *testing.T) {
	t.Parallel()

	mod := discovery.TestModule("svc", "prod", "eu", "app")
	ir := &IR{jobs: []Job{
		{
			name:         "plan-svc-prod-eu-app",
			kind:         JobKindPlan,
			module:       mod,
			env:          map[string]string{"TF_MODULE": "app"},
			dependencies: []JobDependency{{Job: "setup"}},
		},
		{name: "setup", kind: JobKindCommand},
	}}

	plan, ok := ir.JobForModule(JobKindPlan, mod)
	if !ok {
		t.Fatal("JobForModule() did not find plan job")
	}
	if !plan.DependsOnName("setup") {
		t.Fatal("plan job should depend on setup")
	}
	plan.name = "changed"
	plan.env["TF_MODULE"] = "changed"

	fresh, ok := ir.FindJob("plan-svc-prod-eu-app")
	if !ok {
		t.Fatal("FindJob() did not find plan job")
	}
	if got := fresh.Name(); got != "plan-svc-prod-eu-app" {
		t.Fatalf("FindJob() mutation leaked: got %q", got)
	}
	if got := fresh.Env()["TF_MODULE"]; got != "app" {
		t.Fatalf("FindJob() env mutation leaked: got %q", got)
	}
	if got := ir.JobNamesByKind(JobKindPlan); len(got) != 1 || got[0] != "plan-svc-prod-eu-app" {
		t.Fatalf("JobNamesByKind() = %#v", got)
	}
	if !ir.HasDependency("plan-svc-prod-eu-app", "setup") {
		t.Fatal("HasDependency() = false, want true")
	}
}
