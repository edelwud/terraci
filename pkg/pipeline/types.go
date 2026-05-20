package pipeline

import "github.com/edelwud/terraci/pkg/discovery"

// IR is the provider-agnostic intermediate representation of a CI pipeline.
type IR struct {
	Jobs []Job
}

// Job is a single CI job in the IR.
type Job struct {
	Name           string
	Kind           JobKind
	Module         *discovery.Module // nil for command jobs
	Env            map[string]string
	Dependencies   []JobDependency // job edges this depends on
	InputArtifacts []InputArtifact // artifacts restored before this job runs
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

// InputArtifact is an artifact restored from another job before this job runs.
type InputArtifact struct {
	Artifact    Artifact
	ProducerJob string
	Optional    bool
}

// Configured reports whether the artifact has enough data to be published.
func (a Artifact) Configured() bool {
	return a.Name != "" && len(a.Paths) > 0
}

// Configured reports whether the input artifact can be restored.
func (a InputArtifact) Configured() bool {
	return a.ProducerJob != "" && a.Artifact.Configured()
}

// JobDependency is a directed control edge.
type JobDependency struct {
	Job string
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

// Contribution describes provider-independent DAG jobs added by a plugin.
type Contribution struct {
	jobs []ContributedJob
}

// ContributedJob is a command job contributed to the pipeline DAG.
type ContributedJob struct {
	name         string
	commands     []string
	dependencies []JobDependency
	consumes     []ResourceRequest
	produces     []ResourceSpec
	allowFailure bool
}
