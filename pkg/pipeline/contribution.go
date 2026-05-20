package pipeline

import (
	"errors"
	"fmt"
	"strings"
)

// ContributedJobOptions describes a validated command job contributed by a
// plugin to the provider-independent pipeline DAG.
type ContributedJobOptions struct {
	Name         string
	Commands     []string
	Dependencies []JobDependency
	Consumes     []ResourceRequest
	Produces     []ResourceSpec
	AllowFailure bool
}

// PluginCommandJobOptions is the canonical producer-facing builder input for
// plugin-owned command jobs.
type PluginCommandJobOptions struct {
	Name         string
	Commands     []string
	Dependencies []JobDependency
	Consumes     []ResourceRequest
	Produces     []ResourceSpec
	AllowFailure bool
}

// NewContribution builds a contribution value object from validated jobs.
func NewContribution(jobs ...ContributedJob) (*Contribution, error) {
	seenNames := make(map[string]struct{}, len(jobs))
	seenOutputs := make(map[ResourceRef]struct{})
	for i, job := range jobs {
		if err := validateContributedJob(job); err != nil {
			return nil, fmt.Errorf("jobs[%d]: %w", i, err)
		}
		if _, exists := seenNames[job.name]; exists {
			return nil, fmt.Errorf("duplicate contributed job name %q", job.name)
		}
		seenNames[job.name] = struct{}{}

		for _, output := range job.produces {
			if _, exists := seenOutputs[output.Ref]; exists {
				return nil, fmt.Errorf("duplicate produced resource %s", describeResourceRef(output.Ref))
			}
			seenOutputs[output.Ref] = struct{}{}
		}
	}

	clone := make([]ContributedJob, len(jobs))
	for i, job := range jobs {
		clone[i] = job.clone()
	}
	return &Contribution{jobs: clone}, nil
}

// NewPluginCommandJob builds a plugin command job with the same validation as
// NewContributedJob while making the producer-facing intent explicit.
func NewPluginCommandJob(opts PluginCommandJobOptions) (ContributedJob, error) {
	return NewContributedJob(ContributedJobOptions(opts))
}

// NewContributedJob builds a contributed job value object.
func NewContributedJob(opts ContributedJobOptions) (ContributedJob, error) {
	job := ContributedJob{
		name:         strings.TrimSpace(opts.Name),
		commands:     append([]string(nil), opts.Commands...),
		dependencies: append([]JobDependency(nil), opts.Dependencies...),
		consumes:     append([]ResourceRequest(nil), opts.Consumes...),
		produces:     append([]ResourceSpec(nil), opts.Produces...),
		allowFailure: opts.AllowFailure,
	}
	if err := validateContributedJob(job); err != nil {
		return ContributedJob{}, err
	}
	return job, nil
}

// Clone returns a deep copy of c. Nil receivers return nil.
func (c *Contribution) Clone() *Contribution {
	if c == nil {
		return nil
	}
	clone := make([]ContributedJob, len(c.jobs))
	for i, job := range c.jobs {
		clone[i] = job.clone()
	}
	return &Contribution{jobs: clone}
}

// Jobs returns the contributed jobs in declaration order.
func (c *Contribution) Jobs() []ContributedJob {
	if c == nil {
		return nil
	}
	clone := make([]ContributedJob, len(c.jobs))
	for i, job := range c.jobs {
		clone[i] = job.clone()
	}
	return clone
}

// Name returns the job name.
func (j ContributedJob) Name() string { return j.name }

// Commands returns a defensive copy of the command list.
func (j ContributedJob) Commands() []string {
	return append([]string(nil), j.commands...)
}

// Dependencies returns a defensive copy of control dependencies.
func (j ContributedJob) Dependencies() []JobDependency {
	return append([]JobDependency(nil), j.dependencies...)
}

// Consumes returns a defensive copy of resource requests.
func (j ContributedJob) Consumes() []ResourceRequest {
	return append([]ResourceRequest(nil), j.consumes...)
}

// Produces returns a defensive copy of produced resources.
func (j ContributedJob) Produces() []ResourceSpec {
	return append([]ResourceSpec(nil), j.produces...)
}

// AllowFailure reports whether the pipeline job is allowed to fail.
func (j ContributedJob) AllowFailure() bool { return j.allowFailure }

func (j ContributedJob) clone() ContributedJob {
	j.commands = append([]string(nil), j.commands...)
	j.dependencies = append([]JobDependency(nil), j.dependencies...)
	j.consumes = append([]ResourceRequest(nil), j.consumes...)
	j.produces = append([]ResourceSpec(nil), j.produces...)
	return j
}

func validateContributedJob(job ContributedJob) error {
	if job.name == "" {
		return errors.New("name is required")
	}
	if len(job.commands) == 0 {
		return fmt.Errorf("job %q requires at least one command", job.name)
	}
	for i, command := range job.commands {
		if strings.TrimSpace(command) == "" {
			return fmt.Errorf("job %q commands[%d] is empty", job.name, i)
		}
	}
	for i, dep := range job.dependencies {
		if strings.TrimSpace(dep.Job) == "" {
			return fmt.Errorf("job %q dependencies[%d] is empty", job.name, i)
		}
	}
	for i, request := range job.consumes {
		if err := validateResourceRequest(request); err != nil {
			return fmt.Errorf("job %q consumes[%d]: %w", job.name, i, err)
		}
	}

	seenOutputs := make(map[ResourceRef]struct{}, len(job.produces))
	for i, resource := range job.produces {
		if err := validateProducedResource(resource); err != nil {
			return fmt.Errorf("job %q produces[%d]: %w", job.name, i, err)
		}
		if _, exists := seenOutputs[resource.Ref]; exists {
			return fmt.Errorf("job %q produces duplicate resource %s", job.name, describeResourceRef(resource.Ref))
		}
		seenOutputs[resource.Ref] = struct{}{}
	}
	return nil
}

func validateProducedResource(resource ResourceSpec) error {
	if err := validateResourceRef(resource.Ref); err != nil {
		return err
	}
	if resource.Path == "" {
		return fmt.Errorf("%s path is required", describeResourceRef(resource.Ref))
	}
	if err := ValidateWorkspacePath(resource.Path); err != nil {
		return fmt.Errorf("%s path invalid: %w", describeResourceRef(resource.Ref), err)
	}
	return nil
}

func describeResourceRef(ref ResourceRef) string {
	switch {
	case isPlanResourceKind(ref.Kind):
		return fmt.Sprintf("%s:%s", ref.Kind, ref.ModulePath)
	case isPluginResourceKind(ref.Kind):
		return fmt.Sprintf("%s:%s", ref.Kind, ref.Producer)
	default:
		return string(ref.Kind)
	}
}
