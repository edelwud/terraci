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

func (DefaultScheduler) Schedule(ir *pipeline.IR) ([]JobGroup, error) {
	if ir == nil {
		return nil, nil
	}

	var groups []JobGroup
	appendPhaseGroup := func(phase pipeline.Phase) error {
		jobs := ir.JobsByPhase(phase)
		if len(jobs) == 0 {
			return nil
		}
		layers, err := dependencyLayers(jobs)
		if err != nil {
			return fmt.Errorf("schedule phase %s: %w", phase, err)
		}
		for idx, layer := range layers {
			groupName := string(phase)
			if idx > 0 {
				groupName = levelGroupName(groupName, idx)
			}
			groups = append(groups, JobGroup{Name: groupName, Jobs: layer})
		}
		return nil
	}

	if err := appendPhaseGroup(pipeline.PhasePrePlan); err != nil {
		return nil, err
	}
	for levelIdx := range ir.Levels {
		if jobs := ir.PlanJobsForLevel(levelIdx); len(jobs) > 0 {
			groups = append(groups, JobGroup{Name: levelGroupName("plan", levelIdx), Jobs: jobs})
		}
	}
	if err := appendPhaseGroup(pipeline.PhasePostPlan); err != nil {
		return nil, err
	}
	if err := appendPhaseGroup(pipeline.PhasePreApply); err != nil {
		return nil, err
	}
	for levelIdx := range ir.Levels {
		if jobs := ir.ApplyJobsForLevel(levelIdx); len(jobs) > 0 {
			groups = append(groups, JobGroup{Name: levelGroupName("apply", levelIdx), Jobs: jobs})
		}
	}
	if err := appendPhaseGroup(pipeline.PhasePostApply); err != nil {
		return nil, err
	}
	if err := appendPhaseGroup(pipeline.PhaseFinalize); err != nil {
		return nil, err
	}

	// A finalize group is always recorded, even when empty, so that progress
	// reporters and event sinks can render a "finalize: no contributions"
	// stage marker rather than silently skipping the phase boundary.
	if len(ir.JobsByPhase(pipeline.PhaseFinalize)) == 0 {
		groups = append(groups, JobGroup{Name: string(pipeline.PhaseFinalize)})
	}

	return groups, nil
}

func levelGroupName(kind string, index int) string {
	return fmt.Sprintf("%s-level-%d", kind, index)
}

func dependencyLayers(jobs []*pipeline.Job) ([][]*pipeline.Job, error) {
	if len(jobs) == 0 {
		return nil, nil
	}

	pending := make(map[string]*pipeline.Job, len(jobs))
	for _, job := range jobs {
		if job == nil {
			continue
		}
		if _, dup := pending[job.Name]; dup {
			return nil, fmt.Errorf("duplicate job name %q in scheduling phase", job.Name)
		}
		pending[job.Name] = job
	}
	if len(pending) == 0 {
		return nil, nil
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
			// Empty layer with pending jobs means a cycle (or unresolvable
			// dependency among the same phase). Surface as error instead of
			// silently flattening the remainder — that hid execution-order bugs.
			remaining := make([]string, 0, len(pending))
			for name := range pending {
				remaining = append(remaining, name)
			}
			return nil, fmt.Errorf("cycle or unresolvable dependency among jobs: %v", remaining)
		}
		for _, job := range layer {
			delete(pending, job.Name)
		}
		layers = append(layers, layer)
	}

	return layers, nil
}

func hasPendingDependency(job *pipeline.Job, pending map[string]*pipeline.Job) bool {
	for _, dep := range job.Dependencies {
		if pending[dep] != nil {
			return true
		}
	}
	return false
}
