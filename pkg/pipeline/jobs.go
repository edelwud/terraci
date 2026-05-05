package pipeline

import "github.com/edelwud/terraci/pkg/discovery"

// ModuleCount returns the total number of module slots across all levels.
// Counts both plan and apply slots as one — i.e. modules, not jobs.
// Convenience method to replace the duplicated countModules helpers in
// provider plugins (gitlab, github).
func (ir *IR) ModuleCount() int {
	if ir == nil {
		return 0
	}
	count := 0
	for _, level := range ir.Levels {
		count += len(level.Modules)
	}
	return count
}

// JobKind identifies where a job came from inside the pipeline IR.
type JobKind int

const (
	JobKindContributed JobKind = iota
	JobKindPlan
	JobKindApply
)

// NamePrefix returns the canonical prefix used in JobName for module jobs of
// this kind. Contributed jobs carry their own name and return "".
func (k JobKind) NamePrefix() string {
	switch k {
	case JobKindContributed:
		return ""
	case JobKindPlan:
		return "plan"
	case JobKindApply:
		return "apply"
	default:
		return ""
	}
}

// JobRef is a stable traversal view over every executable job in an IR.
type JobRef struct {
	Kind   JobKind
	Job    *Job
	Level  int
	Module *discovery.Module
}

// JobRefs returns every executable job in deterministic IR order.
func (ir *IR) JobRefs() []JobRef {
	if ir == nil {
		return nil
	}

	refs := make([]JobRef, 0, len(ir.Jobs))
	for i := range ir.Jobs {
		refs = append(refs, JobRef{
			Kind: JobKindContributed,
			Job:  &ir.Jobs[i],
		})
	}
	for levelIdx := range ir.Levels {
		level := &ir.Levels[levelIdx]
		for moduleIdx := range level.Modules {
			moduleJobs := &level.Modules[moduleIdx]
			if moduleJobs.Plan != nil {
				refs = append(refs, JobRef{
					Kind:   JobKindPlan,
					Job:    moduleJobs.Plan,
					Level:  level.Index,
					Module: moduleJobs.Module,
				})
			}
			if moduleJobs.Apply != nil {
				refs = append(refs, JobRef{
					Kind:   JobKindApply,
					Job:    moduleJobs.Apply,
					Level:  level.Index,
					Module: moduleJobs.Module,
				})
			}
		}
	}

	return refs
}

// JobNames returns names for every executable job in deterministic IR order.
func (ir *IR) JobNames() []string {
	refs := ir.JobRefs()
	names := make([]string, 0, len(refs))
	for i := range refs {
		if refs[i].Job != nil {
			names = append(names, refs[i].Job.Name)
		}
	}
	return names
}

// AllPlanNames returns names of all plan jobs across all levels.
func (ir *IR) AllPlanNames() []string {
	refs := ir.JobRefs()
	names := make([]string, 0, len(refs))
	for i := range refs {
		if refs[i].Kind == JobKindPlan && refs[i].Job != nil {
			names = append(names, refs[i].Job.Name)
		}
	}
	return names
}

// ContributedJobNames returns names of all contributed jobs.
func (ir *IR) ContributedJobNames() []string {
	if ir == nil {
		return nil
	}

	names := make([]string, 0, len(ir.Jobs))
	for i := range ir.Jobs {
		names = append(names, ir.Jobs[i].Name)
	}
	return names
}

// JobsByPhase returns standalone contributed jobs whose Phase matches.
// The result slice is freshly allocated and safe to mutate.
func (ir *IR) JobsByPhase(phase Phase) []*Job {
	if ir == nil {
		return nil
	}
	var jobs []*Job
	for i := range ir.Jobs {
		if ir.Jobs[i].Phase == phase {
			jobs = append(jobs, &ir.Jobs[i])
		}
	}
	return jobs
}

// PlanJobsForLevel returns plan jobs at the given execution level.
func (ir *IR) PlanJobsForLevel(levelIdx int) []*Job {
	return ir.moduleJobsForLevel(levelIdx, true)
}

// ApplyJobsForLevel returns apply jobs at the given execution level.
func (ir *IR) ApplyJobsForLevel(levelIdx int) []*Job {
	return ir.moduleJobsForLevel(levelIdx, false)
}

func (ir *IR) moduleJobsForLevel(levelIdx int, plan bool) []*Job {
	if ir == nil {
		return nil
	}
	for i := range ir.Levels {
		level := &ir.Levels[i]
		if level.Index != levelIdx {
			continue
		}
		var jobs []*Job
		for j := range level.Modules {
			moduleJobs := &level.Modules[j]
			if plan && moduleJobs.Plan != nil {
				jobs = append(jobs, moduleJobs.Plan)
			} else if !plan && moduleJobs.Apply != nil {
				jobs = append(jobs, moduleJobs.Apply)
			}
		}
		return jobs
	}
	return nil
}
