package pipeline

// ModuleCount returns the number of distinct modules represented by jobs.
func (ir *IR) ModuleCount() int {
	if ir == nil {
		return 0
	}
	seen := make(map[string]struct{})
	for i := range ir.jobs {
		mod := ir.jobs[i].module
		if mod == nil {
			continue
		}
		seen[mod.ID()] = struct{}{}
	}
	return len(seen)
}

// JobKind identifies the canonical role of a pipeline job.
type JobKind string

const (
	JobKindPlan    JobKind = "plan"
	JobKindApply   JobKind = "apply"
	JobKindCommand JobKind = "command"
)

// NamePrefix returns the canonical prefix used in module job names for jobs of
// this kind. Contributed jobs carry their own name and return "".
func (k JobKind) NamePrefix() string {
	switch k {
	case JobKindPlan:
		return "plan"
	case JobKindApply:
		return "apply"
	case JobKindCommand:
		return ""
	default:
		return ""
	}
}

func (k JobKind) valid() bool {
	switch k {
	case JobKindPlan, JobKindApply, JobKindCommand:
		return true
	default:
		return false
	}
}
