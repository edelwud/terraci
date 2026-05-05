package pipeline

import "github.com/edelwud/terraci/pkg/discovery"

// Phase defines when something runs in the pipeline lifecycle.
//
// Values are the canonical stage names: providers can use them directly as
// YAML stage labels, schedulers can use them as group identifiers, and
// switch-statements can rely on string equality. There is intentionally no
// parallel `Stage*` string-constant table — that produced two sources of
// truth for the same lifecycle marker.
type Phase string

const (
	PhasePrePlan   Phase = "pre-plan"   // before terraform plan
	PhasePostPlan  Phase = "post-plan"  // after terraform plan
	PhasePreApply  Phase = "pre-apply"  // before terraform apply
	PhasePostApply Phase = "post-apply" // after terraform apply
	PhaseFinalize  Phase = "finalize"   // after everything — reports, notifications
)

// String returns the stage name for this phase.
func (p Phase) String() string { return string(p) }

// IR is the provider-agnostic intermediate representation of a CI pipeline.
type IR struct {
	Levels []Level
	Jobs   []Job // jobs contributed by feature contributors
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
//
// To distinguish plan / apply / contributed jobs, callers should branch on
// Operation.Type for runtime dispatch and on `Module == nil` to detect
// contributed jobs. There used to be a separate JobType field with iota
// values that defaulted to "plan" for contributed jobs — that zero-value
// trap is the reason it was removed.
type Job struct {
	Name          string
	Phase         Phase             // for contributed jobs: when they run
	Module        *discovery.Module // nil for contributed jobs
	Env           map[string]string
	Dependencies  []string // job names this depends on
	ArtifactPaths []string
	AllowFailure  bool
	Steps         []Step // pre/post steps from contributors
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

// Contribution describes additional steps and standalone jobs that an
// external contributor wants to splice into the generated pipeline.
type Contribution struct {
	// Steps are injected into each module's plan/apply jobs.
	Steps []Step
	// Jobs are standalone jobs added to the pipeline.
	Jobs []ContributedJob
}

// ContributedJob is a standalone job contributed to the pipeline.
type ContributedJob struct {
	Name          string
	Phase         Phase // when it runs; Phase.String() gives the stage name
	Commands      []string
	ArtifactPaths []string
	DependsOnPlan bool
	AllowFailure  bool
}
