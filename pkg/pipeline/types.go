package pipeline

import "github.com/edelwud/terraci/pkg/discovery"

// Phase defines when something runs in the pipeline lifecycle.
type Phase int

const (
	PhasePrePlan   Phase = iota // before terraform plan
	PhasePostPlan               // after terraform plan
	PhasePreApply               // before terraform apply
	PhasePostApply              // after terraform apply
	PhaseFinalize               // after everything — summary, notifications

	phaseFinalizeName = "summary"
)

// String returns the stage name for this phase (e.g., "pre-plan", "post-apply").
func (p Phase) String() string {
	switch p {
	case PhasePrePlan:
		return "pre-plan"
	case PhasePostPlan:
		return "post-plan"
	case PhasePreApply:
		return "pre-apply"
	case PhasePostApply:
		return "post-apply"
	case PhaseFinalize:
		return phaseFinalizeName
	default:
		return "unknown"
	}
}

// JobType distinguishes plan from apply jobs.
type JobType int

const (
	JobTypePlan JobType = iota
	JobTypeApply
)

// IR is the provider-agnostic intermediate representation of a CI pipeline.
type IR struct {
	Levels []Level
	Jobs   []Job // contributed jobs from plugins
}

// AllPlanNames returns names of all plan jobs across all levels.
func (ir *IR) AllPlanNames() []string {
	var names []string
	for _, level := range ir.Levels {
		for _, mj := range level.Modules {
			if mj.Plan != nil {
				names = append(names, mj.Plan.Name)
			}
		}
	}
	return names
}

// ContributedJobNames returns names of all contributed jobs.
func (ir *IR) ContributedJobNames() []string {
	names := make([]string, len(ir.Jobs))
	for i := range ir.Jobs {
		names[i] = ir.Jobs[i].Name
	}
	return names
}

// Level groups modules that can execute in parallel.
type Level struct {
	Index   int
	Modules []ModuleJobs
}

// ModuleJobs holds the plan and apply jobs for a single module.
type ModuleJobs struct {
	Module *discovery.Module
	Plan   *Job // nil if plan disabled
	Apply  *Job // nil if plan-only mode
}

// Job is a single CI job in the IR.
type Job struct {
	Name          string
	Type          JobType
	Phase         Phase             // for contributed jobs: when they run
	Module        *discovery.Module // nil for contributed/summary jobs
	Env           map[string]string
	Dependencies  []string // job names this depends on
	Script        []string // commands to run
	ArtifactPaths []string
	AllowFailure  bool
	Steps         []Step // pre/post steps from plugins
}

// Step is an injected command at a specific phase.
type Step struct {
	Phase   Phase
	Name    string
	Command string
}

// Contribution is what a PipelineContributor plugin provides.
type Contribution struct {
	// Steps are injected into each module's plan/apply jobs.
	Steps []Step
	// Jobs are standalone jobs added to the pipeline.
	Jobs []ContributedJob
}

// ContributedJob is a standalone job from a plugin.
type ContributedJob struct {
	Name          string
	Phase         Phase // when it runs; Phase.String() gives the stage name
	Commands      []string
	ArtifactPaths []string
	DependsOnPlan bool
	AllowFailure  bool
}
