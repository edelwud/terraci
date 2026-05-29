package domain

import (
	"errors"
	"fmt"
	"sort"
)

// Pipeline represents a GitLab CI pipeline.
type Pipeline struct {
	stages    []string
	variables map[string]string
	defaults  *DefaultConfig
	jobs      map[string]Job
	workflow  *Workflow
}

type PipelineOptions struct {
	Stages    []string
	Variables map[string]string
	Default   *DefaultConfig
	Workflow  *Workflow
}

type PipelineBuilder struct {
	opts PipelineOptions
	jobs map[string]Job
}

func NewPipelineBuilder(opts PipelineOptions) *PipelineBuilder {
	return &PipelineBuilder{
		opts: PipelineOptions{
			Stages:    append([]string(nil), opts.Stages...),
			Variables: cloneStringMap(opts.Variables),
			Default:   cloneDefaultConfig(opts.Default),
			Workflow:  cloneWorkflow(opts.Workflow),
		},
		jobs: make(map[string]Job),
	}
}

func EmptyPipeline() *Pipeline {
	return &Pipeline{jobs: make(map[string]Job)}
}

func (b *PipelineBuilder) AddJob(name string, job Job) error {
	if b == nil {
		return errors.New("gitlab pipeline builder is nil")
	}
	if name == "" {
		return errors.New("gitlab job name is required")
	}
	if _, exists := b.jobs[name]; exists {
		return fmt.Errorf("duplicate gitlab job %q", name)
	}
	b.jobs[name] = job.clone()
	return nil
}

func (b *PipelineBuilder) Build() (*Pipeline, error) {
	if b == nil {
		return nil, errors.New("gitlab pipeline builder is nil")
	}
	jobs := make(map[string]Job, len(b.jobs))
	for name := range b.jobs {
		job := b.jobs[name]
		jobs[name] = job.clone()
	}
	return &Pipeline{
		stages:    append([]string(nil), b.opts.Stages...),
		variables: cloneStringMap(b.opts.Variables),
		defaults:  cloneDefaultConfig(b.opts.Default),
		jobs:      jobs,
		workflow:  cloneWorkflow(b.opts.Workflow),
	}, nil
}

func (p *Pipeline) Stages() []string {
	if p == nil {
		return nil
	}
	return append([]string(nil), p.stages...)
}

func (p *Pipeline) Variables() map[string]string {
	if p == nil {
		return nil
	}
	return cloneStringMap(p.variables)
}

func (p *Pipeline) Default() *DefaultConfig {
	if p == nil {
		return nil
	}
	return cloneDefaultConfig(p.defaults)
}

func (p *Pipeline) Workflow() *Workflow {
	if p == nil {
		return nil
	}
	return cloneWorkflow(p.workflow)
}

func (p *Pipeline) Job(name string) (Job, bool) {
	if p == nil {
		return Job{}, false
	}
	job, ok := p.jobs[name]
	if !ok {
		return Job{}, false
	}
	return job.clone(), true
}

func (p *Pipeline) JobNames() []string {
	if p == nil {
		return nil
	}
	names := make([]string, 0, len(p.jobs))
	for name := range p.jobs {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (p *Pipeline) JobCount() int {
	if p == nil {
		return 0
	}
	return len(p.jobs)
}

func (p *Pipeline) HasNeed(jobName, dependency string) bool {
	job, ok := p.Job(jobName)
	return ok && job.HasNeed(dependency)
}
