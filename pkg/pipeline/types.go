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
	ArtifactPaths []string
	AllowFailure  bool
	Steps         []Step // pre/post steps from plugins
	Operation     Operation
}

// Step is an injected command at a specific phase.
type Step struct {
	Phase   Phase
	Name    string
	Command string
}

// OperationType identifies the executable job payload.
type OperationType string

const (
	OperationTypeTerraformPlan  OperationType = "terraform_plan"
	OperationTypeTerraformApply OperationType = "terraform_apply"
	OperationTypeCommands       OperationType = "commands"
)

// Operation describes an executable payload for a pipeline job.
type Operation struct {
	Type      OperationType
	Terraform *TerraformOperation
	Commands  []string
}

// TerraformOperation describes a terraform/tofu operation in a module.
type TerraformOperation struct {
	Kind         OperationType
	ModulePath   string
	InitEnabled  bool
	PlanFile     string
	PlanTextFile string
	PlanJSONFile string
	DetailedPlan bool
	UsePlanFile  bool
	AutoApprove  bool
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
