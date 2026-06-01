package generate

import (
	"reflect"
	"testing"

	"github.com/edelwud/terraci/pkg/execution"
	"github.com/edelwud/terraci/pkg/pipeline"
	"github.com/edelwud/terraci/pkg/pipeline/pipelinetest"
	configpkg "github.com/edelwud/terraci/plugins/gitlab/internal/config"
)

func TestStagePlannerUsesDAGLayers(t *testing.T) {
	t.Parallel()

	planner := newStagePlanner(newSettings(&configpkg.Config{}, execution.Config{}))
	ir := pipelinetest.MustCommandIR(t,
		pipeline.ContributedJobOptions{Name: "plan-0", Commands: []string{"plan-0"}},
		pipeline.ContributedJobOptions{Name: "apply-0", Commands: []string{"apply-0"}, Dependencies: []pipeline.JobDependency{{Job: "plan-0"}}},
		pipeline.ContributedJobOptions{Name: "plan-1", Commands: []string{"plan-1"}, Dependencies: []pipeline.JobDependency{{Job: "apply-0"}}},
		pipeline.ContributedJobOptions{Name: "apply-1", Commands: []string{"apply-1"}, Dependencies: []pipeline.JobDependency{{Job: "plan-1"}}},
		pipeline.ContributedJobOptions{Name: "policy-check", Commands: []string{"policy-check"}, Dependencies: []pipeline.JobDependency{{Job: "plan-1"}}},
		pipeline.ContributedJobOptions{Name: "summary", Commands: []string{"summary"}, Dependencies: []pipeline.JobDependency{{Job: "policy-check"}, {Job: "apply-1"}}},
	)

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
