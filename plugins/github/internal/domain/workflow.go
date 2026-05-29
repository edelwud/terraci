package domain

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"sort"

	"go.yaml.in/yaml/v4"
)

type Workflow struct {
	name        string
	on          WorkflowTrigger
	permissions map[string]string
	env         map[string]string
	concurrency *Concurrency
	jobs        map[string]Job
}

type WorkflowOptions struct {
	Name        string
	On          WorkflowTrigger
	Permissions map[string]string
	Env         map[string]string
	Concurrency *Concurrency
	Jobs        []NamedJob
}

type NamedJob struct {
	Name string
	Job  Job
}

type WorkflowBuilder struct {
	opts WorkflowOptions
	jobs map[string]Job
}

func NewWorkflow(opts WorkflowOptions) (*Workflow, error) {
	builder := NewWorkflowBuilder(WorkflowOptions{
		Name:        opts.Name,
		On:          opts.On,
		Permissions: opts.Permissions,
		Env:         opts.Env,
		Concurrency: opts.Concurrency,
	})
	for i := range opts.Jobs {
		job := opts.Jobs[i]
		if err := builder.AddJob(job.Name, job.Job); err != nil {
			return nil, err
		}
	}
	return builder.Build()
}

func EmptyWorkflow() *Workflow {
	return &Workflow{jobs: make(map[string]Job)}
}

func NewWorkflowBuilder(opts WorkflowOptions) *WorkflowBuilder {
	return &WorkflowBuilder{
		opts: WorkflowOptions{
			Name:        opts.Name,
			On:          cloneWorkflowTrigger(opts.On),
			Permissions: cloneStringMap(opts.Permissions),
			Env:         cloneStringMap(opts.Env),
			Concurrency: cloneConcurrency(opts.Concurrency),
		},
		jobs: make(map[string]Job),
	}
}

func (b *WorkflowBuilder) AddJob(name string, job Job) error {
	if b == nil {
		return errors.New("github workflow builder is nil")
	}
	if name == "" {
		return errors.New("github job name is required")
	}
	if _, exists := b.jobs[name]; exists {
		return fmt.Errorf("duplicate github job %q", name)
	}
	b.jobs[name] = job.clone()
	return nil
}

func (b *WorkflowBuilder) Build() (*Workflow, error) {
	if b == nil {
		return nil, errors.New("github workflow builder is nil")
	}
	jobs := make(map[string]Job, len(b.jobs))
	for name := range b.jobs {
		job := b.jobs[name]
		jobs[name] = job.clone()
	}
	return &Workflow{
		name:        b.opts.Name,
		on:          cloneWorkflowTrigger(b.opts.On),
		permissions: cloneStringMap(b.opts.Permissions),
		env:         cloneStringMap(b.opts.Env),
		concurrency: cloneConcurrency(b.opts.Concurrency),
		jobs:        jobs,
	}, nil
}

func (w *Workflow) Name() string {
	if w == nil {
		return ""
	}
	return w.name
}

func (w *Workflow) On() WorkflowTrigger {
	if w == nil {
		return WorkflowTrigger{}
	}
	return cloneWorkflowTrigger(w.on)
}

func (w *Workflow) Permissions() map[string]string {
	if w == nil {
		return nil
	}
	return cloneStringMap(w.permissions)
}

func (w *Workflow) Env() map[string]string {
	if w == nil {
		return nil
	}
	return cloneStringMap(w.env)
}

func (w *Workflow) Concurrency() *Concurrency {
	if w == nil {
		return nil
	}
	return cloneConcurrency(w.concurrency)
}

func (w *Workflow) Job(name string) (Job, bool) {
	if w == nil {
		return Job{}, false
	}
	job, ok := w.jobs[name]
	if !ok {
		return Job{}, false
	}
	return job.clone(), true
}

func (w *Workflow) Jobs() map[string]Job {
	if w == nil {
		return nil
	}
	out := make(map[string]Job, len(w.jobs))
	for name := range w.jobs {
		job := w.jobs[name]
		out[name] = job.clone()
	}
	return out
}

func (w *Workflow) JobNames() []string {
	if w == nil {
		return nil
	}
	names := make([]string, 0, len(w.jobs))
	for name := range w.jobs {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (w *Workflow) JobCount() int {
	if w == nil {
		return 0
	}
	return len(w.jobs)
}

func (w *Workflow) HasNeed(jobName, dependency string) bool {
	job, ok := w.Job(jobName)
	return ok && job.HasNeed(dependency)
}

type WorkflowTrigger struct {
	Push             *PushTrigger `yaml:"push,omitempty"`
	PullRequest      *PRTrigger   `yaml:"pull_request,omitempty"`
	WorkflowDispatch any          `yaml:"workflow_dispatch,omitempty"`
}

type PushTrigger struct {
	Branches []string `yaml:"branches,omitempty"`
}

type PRTrigger struct {
	Branches []string `yaml:"branches,omitempty"`
}

type Concurrency struct {
	Group            string `yaml:"group"`
	CancelInProgress bool   `yaml:"cancel-in-progress"`
}

type JobOptions struct {
	Name        string
	RunsOn      string
	Container   *Container
	Needs       []string
	If          string
	Environment string
	Concurrency *Concurrency
	Env         map[string]string
	Steps       []Step
}

type Job struct {
	name        string
	runsOn      string
	container   *Container
	needs       []string
	ifExpr      string
	environment string
	concurrency *Concurrency
	env         map[string]string
	steps       []Step
}

type Container struct {
	Image string            `yaml:"image"`
	Env   map[string]string `yaml:"env,omitempty"`
}

type StepOptions struct {
	Name            string
	Uses            string
	With            map[string]string
	Run             string
	Env             map[string]string
	If              string
	ContinueOnError bool
}

type Step struct {
	name            string
	uses            string
	with            map[string]string
	run             string
	env             map[string]string
	ifExpr          string
	continueOnError bool
}

func NewJob(opts JobOptions) (Job, error) {
	if opts.RunsOn == "" {
		return Job{}, errors.New("github job runs-on is required")
	}
	if len(opts.Steps) == 0 {
		return Job{}, errors.New("github job steps are required")
	}
	return Job{
		name:        opts.Name,
		runsOn:      opts.RunsOn,
		container:   cloneContainer(opts.Container),
		needs:       append([]string(nil), opts.Needs...),
		ifExpr:      opts.If,
		environment: opts.Environment,
		concurrency: cloneConcurrency(opts.Concurrency),
		env:         cloneStringMap(opts.Env),
		steps:       cloneSteps(opts.Steps),
	}, nil
}

func (j Job) Name() string { return j.name }

func (j Job) RunsOn() string { return j.runsOn }

func (j Job) Container() *Container { return cloneContainer(j.container) }

func (j Job) Needs() []string { return append([]string(nil), j.needs...) }

func (j Job) HasNeed(name string) bool {
	return slices.Contains(j.needs, name)
}

func (j Job) If() string { return j.ifExpr }

func (j Job) Environment() string { return j.environment }

func (j Job) Concurrency() *Concurrency { return cloneConcurrency(j.concurrency) }

func (j Job) Env() map[string]string { return cloneStringMap(j.env) }

func (j Job) Steps() []Step { return cloneSteps(j.steps) }

func (j Job) clone() Job {
	return Job{
		name:        j.name,
		runsOn:      j.runsOn,
		container:   cloneContainer(j.container),
		needs:       append([]string(nil), j.needs...),
		ifExpr:      j.ifExpr,
		environment: j.environment,
		concurrency: cloneConcurrency(j.concurrency),
		env:         cloneStringMap(j.env),
		steps:       cloneSteps(j.steps),
	}
}

func (j Job) MarshalYAML() (any, error) {
	return struct {
		Name        string            `yaml:"name,omitempty"`
		RunsOn      string            `yaml:"runs-on"`
		Container   *Container        `yaml:"container,omitempty"`
		Needs       []string          `yaml:"needs,omitempty"`
		If          string            `yaml:"if,omitempty"`
		Environment string            `yaml:"environment,omitempty"`
		Concurrency *Concurrency      `yaml:"concurrency,omitempty"`
		Env         map[string]string `yaml:"env,omitempty"`
		Steps       []Step            `yaml:"steps"`
	}{
		Name:        j.name,
		RunsOn:      j.runsOn,
		Container:   cloneContainer(j.container),
		Needs:       append([]string(nil), j.needs...),
		If:          j.ifExpr,
		Environment: j.environment,
		Concurrency: cloneConcurrency(j.concurrency),
		Env:         cloneStringMap(j.env),
		Steps:       cloneSteps(j.steps),
	}, nil
}

func NewStep(opts StepOptions) Step {
	return Step{
		name:            opts.Name,
		uses:            opts.Uses,
		with:            cloneStringMap(opts.With),
		run:             opts.Run,
		env:             cloneStringMap(opts.Env),
		ifExpr:          opts.If,
		continueOnError: opts.ContinueOnError,
	}
}

func (s Step) Name() string { return s.name }

func (s Step) Uses() string { return s.uses }

func (s Step) With() map[string]string { return cloneStringMap(s.with) }

func (s Step) Run() string { return s.run }

func (s Step) Env() map[string]string { return cloneStringMap(s.env) }

func (s Step) If() string { return s.ifExpr }

func (s Step) ContinueOnError() bool { return s.continueOnError }

func (s Step) clone() Step {
	return NewStep(StepOptions{
		Name:            s.name,
		Uses:            s.uses,
		With:            s.with,
		Run:             s.run,
		Env:             s.env,
		If:              s.ifExpr,
		ContinueOnError: s.continueOnError,
	})
}

func (s Step) MarshalYAML() (any, error) {
	return struct {
		Name            string            `yaml:"name,omitempty"`
		Uses            string            `yaml:"uses,omitempty"`
		With            map[string]string `yaml:"with,omitempty"`
		Run             string            `yaml:"run,omitempty"`
		Env             map[string]string `yaml:"env,omitempty"`
		If              string            `yaml:"if,omitempty"`
		ContinueOnError bool              `yaml:"continue-on-error,omitempty"`
	}{
		Name:            s.name,
		Uses:            s.uses,
		With:            cloneStringMap(s.with),
		Run:             s.run,
		Env:             cloneStringMap(s.env),
		If:              s.ifExpr,
		ContinueOnError: s.continueOnError,
	}, nil
}

func (w *Workflow) ToYAML() ([]byte, error) {
	type workflowYAML struct {
		Name        string            `yaml:"name"`
		On          WorkflowTrigger   `yaml:"on"`
		Permissions map[string]string `yaml:"permissions,omitempty"`
		Env         map[string]string `yaml:"env,omitempty"`
		Concurrency *Concurrency      `yaml:"concurrency,omitempty"`
		Jobs        map[string]Job    `yaml:"jobs"`
	}
	payload := workflowYAML{Jobs: map[string]Job{}}
	if w != nil {
		payload.Name = w.name
		payload.On = cloneWorkflowTrigger(w.on)
		payload.Permissions = cloneStringMap(w.permissions)
		payload.Env = cloneStringMap(w.env)
		payload.Concurrency = cloneConcurrency(w.concurrency)
		for name := range w.jobs {
			job := w.jobs[name]
			payload.Jobs[name] = job.clone()
		}
	}
	data, err := yaml.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal workflow: %w", err)
	}

	header := []byte("# Generated by terraci — do not edit\n")
	return append(header, data...), nil
}

func cloneWorkflowTrigger(in WorkflowTrigger) WorkflowTrigger {
	out := in
	if in.Push != nil {
		out.Push = &PushTrigger{Branches: append([]string(nil), in.Push.Branches...)}
	}
	if in.PullRequest != nil {
		out.PullRequest = &PRTrigger{Branches: append([]string(nil), in.PullRequest.Branches...)}
	}
	return out
}

func cloneConcurrency(in *Concurrency) *Concurrency {
	if in == nil {
		return nil
	}
	return &Concurrency{Group: in.Group, CancelInProgress: in.CancelInProgress}
}

func cloneContainer(in *Container) *Container {
	if in == nil {
		return nil
	}
	return &Container{Image: in.Image, Env: cloneStringMap(in.Env)}
}

func cloneSteps(in []Step) []Step {
	if len(in) == 0 {
		return nil
	}
	out := make([]Step, len(in))
	for i, step := range in {
		out[i] = step.clone()
	}
	return out
}

func cloneStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	maps.Copy(out, in)
	return out
}
