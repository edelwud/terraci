package pipeline

import "github.com/edelwud/terraci/pkg/discovery"

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
	Name           string
	Module         *discovery.Module // nil for contributed jobs
	Env            map[string]string
	Dependencies   []JobDependency // job edges this depends on
	InputArtifacts []Artifact      // artifacts restored before this job runs
	OutputArtifact Artifact
	Consumes       []ResourceSpec
	Produces       []ResourceSpec
	AllowFailure   bool
	Operation      Operation
}

// Artifact is a named CI artifact whose paths must be restored relative to
// the downstream job workspace. Providers may stage files internally, but
// consumers should see each path exactly as listed here.
type Artifact struct {
	Name  string
	Paths []string
}

// Configured reports whether the artifact has enough data to be published.
func (a Artifact) Configured() bool {
	return a.Name != "" && len(a.Paths) > 0
}

// JobDependency is a directed job edge. Artifacts marks dependencies whose
// output artifact must be restored into the downstream workspace.
type JobDependency struct {
	Job       string
	Artifacts bool
	Optional  bool
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
}

// Contribution describes standalone jobs that an external contributor wants to
// add to the generated pipeline.
type Contribution struct {
	// Jobs are standalone jobs added to the pipeline.
	Jobs []ContributedJob
}

// ContributedJob is a standalone job contributed to the pipeline.
type ContributedJob struct {
	Name         string
	Commands     []string
	Dependencies []JobDependency
	Consumes     []ResourceRequest
	Produces     []ResourceSpec
	AllowFailure bool
}
