package generate

import (
	"reflect"
	"testing"

	"github.com/edelwud/terraci/pkg/execution"
	"github.com/edelwud/terraci/pkg/pipeline"
	configpkg "github.com/edelwud/terraci/plugins/gitlab/internal/config"
)

func TestContributionIndexBuildsStageLookup(t *testing.T) {
	index := newContributionIndex([]*pipeline.Contribution{
		{
			Jobs: []pipeline.ContributedJob{
				{Name: "policy-check", Phase: pipeline.PhasePostPlan},
				{Name: "summary", Phase: pipeline.PhaseFinalize},
			},
		},
	})

	if !index.hasContributedJobs() {
		t.Fatal("expected contribution index to report jobs")
	}
	if got := index.stageFor("policy-check"); got != pipeline.PhasePostPlan.String() {
		t.Fatalf("stageFor(policy-check) = %q", got)
	}
	if got := index.stageFor("summary"); got != pipeline.PhaseFinalize.String() {
		t.Fatalf("stageFor(summary) = %q", got)
	}
}

func TestStagePlannerPlacesContributedStagesAfterPlanAndFinalizeLast(t *testing.T) {
	planner := newStagePlanner(
		newSettings(&configpkg.Config{}, execution.Config{
			Binary:      "terraform",
			InitEnabled: true,
			PlanEnabled: true,
			PlanMode:    execution.PlanModeStandard,
			Parallelism: 4,
		}),
		newContributionIndex([]*pipeline.Contribution{
			{
				Jobs: []pipeline.ContributedJob{
					{Name: "policy-check", Phase: pipeline.PhasePostPlan},
					{Name: "summary", Phase: pipeline.PhaseFinalize},
				},
			},
		}),
	)

	ir := &pipeline.IR{
		Levels: []pipeline.Level{{Index: 0}, {Index: 1}},
	}

	got := planner.stages(ir)
	want := []string{
		"deploy-plan-0",
		"deploy-apply-0",
		"deploy-plan-1",
		"post-plan",
		"deploy-apply-1",
		"finalize",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("stages() = %#v, want %#v", got, want)
	}
}

func TestStagePlannerAppendsContributionsWhenNoPlanStagesExist(t *testing.T) {
	planner := newStagePlanner(
		newSettings(&configpkg.Config{}, execution.Config{
			Binary:      "terraform",
			InitEnabled: true,
			PlanEnabled: false,
			PlanMode:    execution.PlanModeStandard,
			Parallelism: 4,
		}),
		newContributionIndex([]*pipeline.Contribution{
			{
				Jobs: []pipeline.ContributedJob{
					{Name: "summary", Phase: pipeline.PhaseFinalize},
				},
			},
		}),
	)

	ir := &pipeline.IR{Levels: []pipeline.Level{{Index: 0}}}
	got := planner.stages(ir)
	want := []string{"deploy-apply-0", "finalize"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("stages() = %#v, want %#v", got, want)
	}
}
