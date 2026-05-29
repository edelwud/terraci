package domain

import (
	"errors"
	"slices"
)

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
