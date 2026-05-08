package pipeline

import "github.com/edelwud/terraci/pkg/discovery"

// ModuleCount returns the total number of module slots across all levels.
// Counts both plan and apply slots as one — i.e. modules, not jobs.
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
// JobRefs is the canonical IR enumeration entry point.
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
	for i := range ir.Jobs {
		refs = append(refs, JobRef{
			Kind: JobKindContributed,
			Job:  &ir.Jobs[i],
		})
	}

	return refs
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
