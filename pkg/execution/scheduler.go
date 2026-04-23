package execution

import (
	"fmt"

	"github.com/edelwud/terraci/pkg/pipeline"
)

// JobGroup is a barriered set of jobs that may run in parallel.
type JobGroup struct {
	Name string
	Jobs []*pipeline.Job
}

// DefaultScheduler schedules pre/post phases around level-based plan/apply groups.
type DefaultScheduler struct{}

func (DefaultScheduler) Schedule(plan *Plan) []JobGroup {
	if plan == nil || plan.IR == nil {
		return nil
	}

	var groups []JobGroup
	appendPhaseGroup := func(name string, phase pipeline.Phase) {
		if jobs := plan.JobsByPhase(phase); len(jobs) > 0 {
			groups = append(groups, JobGroup{Name: name, Jobs: jobs})
		}
	}

	appendPhaseGroup("pre-plan", pipeline.PhasePrePlan)
	for levelIdx := range plan.IR.Levels {
		if jobs := plan.PlanJobsForLevel(levelIdx); len(jobs) > 0 {
			groups = append(groups, JobGroup{Name: levelGroupName("plan", levelIdx), Jobs: jobs})
		}
	}
	appendPhaseGroup("post-plan", pipeline.PhasePostPlan)
	appendPhaseGroup("pre-apply", pipeline.PhasePreApply)
	for levelIdx := range plan.IR.Levels {
		if jobs := plan.ApplyJobsForLevel(levelIdx); len(jobs) > 0 {
			groups = append(groups, JobGroup{Name: levelGroupName("apply", levelIdx), Jobs: jobs})
		}
	}
	appendPhaseGroup("post-apply", pipeline.PhasePostApply)
	groups = append(groups, JobGroup{Name: "finalize", Jobs: plan.JobsByPhase(pipeline.PhaseFinalize)})

	return groups
}

func levelGroupName(kind string, index int) string {
	return fmt.Sprintf("%s-level-%d", kind, index)
}
