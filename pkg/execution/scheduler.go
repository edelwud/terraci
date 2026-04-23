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
		jobs := plan.JobsByPhase(phase)
		if len(jobs) == 0 {
			return
		}
		for idx, layer := range dependencyLayers(jobs) {
			groupName := name
			if idx > 0 {
				groupName = levelGroupName(name, idx)
			}
			groups = append(groups, JobGroup{Name: groupName, Jobs: layer})
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
	appendPhaseGroup("finalize", pipeline.PhaseFinalize)
	if len(plan.JobsByPhase(pipeline.PhaseFinalize)) == 0 {
		groups = append(groups, JobGroup{Name: "finalize"})
	}

	return groups
}

func levelGroupName(kind string, index int) string {
	return fmt.Sprintf("%s-level-%d", kind, index)
}

func dependencyLayers(jobs []*pipeline.Job) [][]*pipeline.Job {
	if len(jobs) == 0 {
		return nil
	}

	pending := make(map[string]*pipeline.Job, len(jobs))
	for _, job := range jobs {
		if job == nil {
			continue
		}
		pending[job.Name] = job
	}
	if len(pending) == 0 {
		return nil
	}

	var layers [][]*pipeline.Job
	for len(pending) > 0 {
		layer := make([]*pipeline.Job, 0, len(pending))
		for _, job := range jobs {
			if job == nil || pending[job.Name] == nil {
				continue
			}
			if hasPendingDependency(job, pending) {
				continue
			}
			layer = append(layer, job)
		}
		if len(layer) == 0 {
			// Cycles or duplicate names are invalid IR, but the scheduler cannot
			// return errors. Preserve old behavior by releasing the remaining jobs
			// in declaration order rather than deadlocking execution.
			for _, job := range jobs {
				if job != nil && pending[job.Name] != nil {
					layer = append(layer, job)
				}
			}
		}
		for _, job := range layer {
			delete(pending, job.Name)
		}
		layers = append(layers, layer)
	}

	return layers
}

func hasPendingDependency(job *pipeline.Job, pending map[string]*pipeline.Job) bool {
	for _, dep := range job.Dependencies {
		if pending[dep] != nil {
			return true
		}
	}
	return false
}
