package execution

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/edelwud/terraci/pkg/pipeline"
)

// JobRunner executes one job.
type JobRunner interface {
	Run(ctx context.Context, job *pipeline.Job) error
}

// EventSink consumes structured execution events.
type EventSink interface {
	JobStarted(job *pipeline.Job)
	JobFinished(job *pipeline.Job, result *JobResult)
}

// Scheduler builds execution groups from a pipeline IR.
//
// Returns an error if the IR is structurally invalid (cycles or duplicate
// job names) — preventing silent fallback that would run dependent jobs in
// the wrong order.
type Scheduler interface {
	Schedule(ir *pipeline.IR) ([]JobGroup, error)
}

// WorkerPool runs a group of jobs with bounded concurrency.
type WorkerPool interface {
	Run(ctx context.Context, jobs []*pipeline.Job, fn func(context.Context, *pipeline.Job) error) error
}

// ExecutorOption mutates executor behavior.
type ExecutorOption func(*Executor)

// Executor executes a Plan.
type Executor struct {
	runner    JobRunner
	scheduler Scheduler
	workers   WorkerPool
	sink      EventSink
}

// NewExecutor constructs an Executor.
func NewExecutor(runner JobRunner, opts ...ExecutorOption) *Executor {
	executor := &Executor{
		runner:    runner,
		scheduler: DefaultScheduler{},
		workers:   boundedWorkerPool{parallelism: 0},
		sink:      noopEventSink{},
	}
	for _, opt := range opts {
		if opt != nil {
			opt(executor)
		}
	}
	return executor
}

// WithScheduler overrides the execution scheduler.
func WithScheduler(s Scheduler) ExecutorOption {
	return func(e *Executor) {
		if s != nil {
			e.scheduler = s
		}
	}
}

// WithParallelism bounds concurrent jobs inside one execution group.
func WithParallelism(parallelism int) ExecutorOption {
	return func(e *Executor) {
		e.workers = boundedWorkerPool{parallelism: parallelism}
	}
}

// WithEventSink configures an execution event sink.
func WithEventSink(sink EventSink) ExecutorOption {
	return func(e *Executor) {
		if sink != nil {
			e.sink = sink
		}
	}
}

// Execute runs the pipeline IR group-by-group.
func (e *Executor) Execute(ctx context.Context, ir *pipeline.IR) (*Result, error) {
	if e == nil || e.runner == nil {
		return nil, errors.New("executor runner is not configured")
	}
	if e.scheduler == nil {
		return nil, errors.New("executor scheduler is not configured")
	}
	if e.workers == nil {
		return nil, errors.New("executor workers are not configured")
	}
	if ir == nil {
		return nil, errors.New("execution IR is nil")
	}
	if err := ir.Validate(); err != nil {
		return nil, fmt.Errorf("invalid execution IR: %w", err)
	}

	result := &Result{}
	var mu sync.Mutex

	record := func(job *pipeline.Job, status JobStatus, started time.Time, err error) *JobResult {
		jobResult := &JobResult{
			Name:       job.Name,
			Status:     status,
			StartedAt:  started,
			FinishedAt: time.Now(),
			Err:        err,
		}
		mu.Lock()
		result.Jobs = append(result.Jobs, jobResult)
		mu.Unlock()
		return jobResult
	}

	groups, err := e.scheduler.Schedule(ir)
	if err != nil {
		return nil, fmt.Errorf("schedule pipeline: %w", err)
	}
	for _, group := range groups {
		result.Groups = append(result.Groups, GroupResult{
			Name:     group.Name,
			JobCount: len(group.Jobs),
		})
		if len(group.Jobs) == 0 {
			continue
		}
		err := e.workers.Run(ctx, group.Jobs, func(runCtx context.Context, job *pipeline.Job) error {
			started := time.Now()
			e.sink.JobStarted(job)
			err := e.runner.Run(runCtx, job)

			status := JobStatusSucceeded
			if err != nil {
				status = JobStatusFailed
			}
			jobResult := record(job, status, started, err)
			e.sink.JobFinished(job, jobResult)
			return err
		})
		if err != nil {
			return result, err
		}
	}

	return result, nil
}

type noopEventSink struct{}

func (noopEventSink) JobStarted(*pipeline.Job)              {}
func (noopEventSink) JobFinished(*pipeline.Job, *JobResult) {}
