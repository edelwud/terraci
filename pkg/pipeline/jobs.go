package pipeline

import "github.com/edelwud/terraci/pkg/discovery"

// JobKind identifies where a job came from inside the pipeline IR.
type JobKind int

const (
	JobKindContributed JobKind = iota
	JobKindPlan
	JobKindApply
)

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
