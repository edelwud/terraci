package generate

import (
	"reflect"
	"testing"

	"github.com/edelwud/terraci/pkg/execution"
	"github.com/edelwud/terraci/pkg/pipeline"
	configpkg "github.com/edelwud/terraci/plugins/gitlab/internal/config"
)

func TestStagePlannerUsesDAGLayers(t *testing.T) {
	t.Parallel()

	planner := newStagePlanner(newSettings(&configpkg.Config{}, execution.Config{PlanEnabled: true}))
	ir := &pipeline.IR{
		Levels: []pipeline.Level{
			{Index: 0, Modules: []pipeline.ModuleJobs{{Plan: &pipeline.Job{Name: "plan-0"}, Apply: &pipeline.Job{Name: "apply-0", Dependencies: []pipeline.JobDependency{{Job: "plan-0"}}}}}},
			{Index: 1, Modules: []pipeline.ModuleJobs{{Plan: &pipeline.Job{Name: "plan-1", Dependencies: []pipeline.JobDependency{{Job: "apply-0"}}}, Apply: &pipeline.Job{Name: "apply-1", Dependencies: []pipeline.JobDependency{{Job: "plan-1"}}}}}},
		},
		Jobs: []pipeline.Job{
			{Name: "policy-check", Dependencies: []pipeline.JobDependency{{Job: "plan-1"}}},
			{Name: "summary", Dependencies: []pipeline.JobDependency{{Job: "policy-check"}, {Job: "apply-1"}}},
		},
	}

	got, err := planner.plan(ir)
	if err != nil {
		t.Fatalf("plan() error = %v", err)
	}
	wantStages := []string{"deploy-0", "deploy-1", "deploy-2", "deploy-3", "deploy-4"}
	if !reflect.DeepEqual(got.stages, wantStages) {
		t.Fatalf("stages = %#v, want %#v", got.stages, wantStages)
	}
	if got.stageByJob["summary"] != "deploy-4" {
		t.Fatalf("summary stage = %q, want deploy-4", got.stageByJob["summary"])
	}
}
