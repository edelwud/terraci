package pipeline

import (
	"maps"

	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/terraformrun"
)

// IR is the provider-agnostic intermediate representation of a CI pipeline.
type IR struct {
	jobs []Job
}

// Job is a single CI job in the IR.
type Job struct {
	name           string
	kind           JobKind
	module         *discovery.Module // nil for command jobs
	env            map[string]string
	dependencies   []JobDependency // job edges this depends on
	inputArtifacts []InputArtifact // artifacts restored before this job runs
	outputArtifact Artifact
	consumes       []ResourceSpec
	produces       []ResourceSpec
	allowFailure   bool
	operation      Operation
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
	typ       OperationType
	terraform *TerraformOperation
	commands  []string
}

// TerraformOperation describes a terraform/tofu operation in a module.
type TerraformOperation struct {
	binary       terraformrun.Binary
	kind         OperationType
	modulePath   string
	initEnabled  bool
	planFile     string
	planTextFile string
	planJSONFile string
	detailedPlan bool
	usePlanFile  bool
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

// Jobs returns the IR jobs in deterministic execution-plan order.
func (ir *IR) Jobs() []Job {
	if ir == nil {
		return nil
	}
	return cloneJobs(ir.jobs)
}

// FindJob returns a defensive copy of the job with name.
func (ir *IR) FindJob(name string) (Job, bool) {
	if ir == nil || name == "" {
		var zero Job
		return zero, false
	}
	for i := range ir.jobs {
		if ir.jobs[i].name == name {
			return ir.jobs[i].clone(), true
		}
	}
	var zero Job
	return zero, false
}

// JobsByKind returns defensive copies of jobs with the requested kind.
func (ir *IR) JobsByKind(kind JobKind) []Job {
	if ir == nil {
		return nil
	}
	jobs := make([]Job, 0)
	for i := range ir.jobs {
		if ir.jobs[i].kind == kind {
			jobs = append(jobs, ir.jobs[i].clone())
		}
	}
	if len(jobs) == 0 {
		return nil
	}
	return jobs
}

// JobNamesByKind returns names of jobs with the requested kind.
func (ir *IR) JobNamesByKind(kind JobKind) []string {
	if ir == nil {
		return nil
	}
	names := make([]string, 0)
	for i := range ir.jobs {
		if ir.jobs[i].kind == kind {
			names = append(names, ir.jobs[i].name)
		}
	}
	if len(names) == 0 {
		return nil
	}
	return names
}

// JobForModule returns the job for a module and kind without exposing
// module-job naming rules to callers.
func (ir *IR) JobForModule(kind JobKind, module *discovery.Module) (Job, bool) {
	if ir == nil || module == nil {
		var zero Job
		return zero, false
	}
	moduleID := module.ID()
	for i := range ir.jobs {
		job := ir.jobs[i]
		if job.kind == kind && job.module != nil && job.module.ID() == moduleID {
			return job.clone(), true
		}
	}
	var zero Job
	return zero, false
}

// HasDependency reports whether jobName depends on dependencyName.
func (ir *IR) HasDependency(jobName, dependencyName string) bool {
	job, ok := ir.FindJob(jobName)
	if !ok {
		return false
	}
	return job.DependsOnName(dependencyName)
}

func cloneJobs(jobs []Job) []Job {
	if len(jobs) == 0 {
		return nil
	}
	clone := make([]Job, len(jobs))
	for i := range jobs {
		clone[i] = jobs[i].clone()
	}
	return clone
}

func newCommandJob(job ContributedJob) Job {
	produces := job.Produces()
	return Job{
		name:           job.Name(),
		kind:           JobKindCommand,
		dependencies:   job.Dependencies(),
		outputArtifact: resultArtifactFromResources(job.Name(), produces),
		produces:       produces,
		allowFailure:   job.AllowFailure(),
		operation:      newCommandOperation(job.Commands()),
	}
}

// Name returns the job name.
func (j Job) Name() string { return j.name }

// Kind returns the canonical job role.
func (j Job) Kind() JobKind { return j.kind }

// Module returns the module associated with Terraform jobs, or nil for command jobs.
func (j Job) Module() *discovery.Module { return j.module }

// Env returns a defensive copy of job environment variables.
func (j Job) Env() map[string]string {
	if len(j.env) == 0 {
		return nil
	}
	return maps.Clone(j.env)
}

// Dependencies returns a defensive copy of control dependencies.
func (j Job) Dependencies() []JobDependency {
	return append([]JobDependency(nil), j.dependencies...)
}

// DependsOn reports whether this job depends on the supplied job.
func (j Job) DependsOn(dep Job) bool {
	if dep.name == "" {
		return false
	}
	return j.DependsOnName(dep.name)
}

// DependsOnName reports whether this job depends on the supplied job name.
func (j Job) DependsOnName(dep string) bool {
	if dep == "" {
		return false
	}
	for _, dependency := range j.dependencies {
		if dependency.Job == dep {
			return true
		}
	}
	return false
}

// InputArtifacts returns a defensive copy of artifacts restored for this job.
func (j Job) InputArtifacts() []InputArtifact {
	return append([]InputArtifact(nil), j.inputArtifacts...)
}

// OutputArtifact returns the artifact produced by this job, if any.
func (j Job) OutputArtifact() Artifact { return cloneArtifact(j.outputArtifact) }

// Consumes returns a defensive copy of consumed resources.
func (j Job) Consumes() []ResourceSpec {
	return append([]ResourceSpec(nil), j.consumes...)
}

// Produces returns a defensive copy of produced resources.
func (j Job) Produces() []ResourceSpec {
	return append([]ResourceSpec(nil), j.produces...)
}

// AllowFailure reports whether the job may fail without failing the pipeline.
func (j Job) AllowFailure() bool { return j.allowFailure }

// Operation returns the executable job payload.
func (j Job) Operation() Operation { return j.operation.clone() }

func (j Job) clone() Job {
	j.env = maps.Clone(j.env)
	j.dependencies = append([]JobDependency(nil), j.dependencies...)
	j.inputArtifacts = append([]InputArtifact(nil), j.inputArtifacts...)
	j.outputArtifact = cloneArtifact(j.outputArtifact)
	j.consumes = append([]ResourceSpec(nil), j.consumes...)
	j.produces = append([]ResourceSpec(nil), j.produces...)
	j.operation = j.operation.clone()
	return j
}

// Type returns the operation type.
func (o Operation) Type() OperationType { return o.typ }

// Terraform returns a defensive copy of the terraform operation, if present.
func (o Operation) Terraform() *TerraformOperation {
	if o.terraform == nil {
		return nil
	}
	clone := *o.terraform
	return &clone
}

// Commands returns a defensive copy of command operation lines.
func (o Operation) Commands() []string {
	return append([]string(nil), o.commands...)
}

func newCommandOperation(commands []string) Operation {
	return Operation{
		typ:      OperationTypeCommands,
		commands: append([]string(nil), commands...),
	}
}

func (o Operation) clone() Operation {
	o.commands = append([]string(nil), o.commands...)
	if o.terraform != nil {
		terraform := *o.terraform
		o.terraform = &terraform
	}
	return o
}

// Kind returns the terraform operation kind.
func (o TerraformOperation) Kind() OperationType { return o.kind }

// Binary returns the Terraform-compatible executable name for this operation.
func (o TerraformOperation) Binary() string { return o.binary.String() }

// ModulePath returns the workspace-relative module path.
func (o TerraformOperation) ModulePath() string { return o.modulePath }

// InitEnabled reports whether terraform init should run before this operation.
func (o TerraformOperation) InitEnabled() bool { return o.initEnabled }

// PlanFile returns the workspace-relative binary plan path.
func (o TerraformOperation) PlanFile() string { return o.planFile }

// PlanTextFile returns the workspace-relative text plan path.
func (o TerraformOperation) PlanTextFile() string { return o.planTextFile }

// PlanJSONFile returns the workspace-relative JSON plan path.
func (o TerraformOperation) PlanJSONFile() string { return o.planJSONFile }

// DetailedPlan reports whether the plan job emits detailed plan artifacts.
func (o TerraformOperation) DetailedPlan() bool { return o.detailedPlan }

// UsePlanFile reports whether apply should consume the binary plan file.
func (o TerraformOperation) UsePlanFile() bool { return o.usePlanFile }

func cloneArtifact(artifact Artifact) Artifact {
	artifact.Paths = append([]string(nil), artifact.Paths...)
	return artifact
}
