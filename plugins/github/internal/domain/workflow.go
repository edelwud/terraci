package domain

import (
	"errors"
	"fmt"
	"sort"
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
}

type WorkflowBuilder struct {
	opts WorkflowOptions
	jobs map[string]Job
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
