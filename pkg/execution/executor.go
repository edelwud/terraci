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
	Run(ctx context.Context, job pipeline.Job) error
}

// EventSink consumes structured execution events.
type EventSink interface {
	JobStarted(event JobEvent)
	JobFinished(event JobEvent, result JobResult)
}

// Scheduler builds execution groups from a pipeline IR.
//
// Returns an error if the IR is structurally invalid (cycles or duplicate
// job names) — preventing silent fallback that would run dependent jobs in
// the wrong order.
type Scheduler interface {
	Schedule(ir *pipeline.IR) ([]pipeline.JobGroup, error)
}

// WorkerPool runs a group of jobs with bounded concurrency.
type WorkerPool interface {
	Run(ctx context.Context, jobs []pipeline.Job, fn func(context.Context, pipeline.Job) error) error
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

	var (
		groupsRecorded []GroupResult
		jobOrder       []string
		jobsRecorded   = make(map[string]JobResult)
	)
	var mu sync.Mutex

	record := func(job pipeline.Job, status JobStatus, started, finished time.Time, err error) (JobResult, error) {
		jobResult, buildErr := NewJobResult(JobResultOptions{
			Name:              job.Name(),
			Status:            status,
			StartedAt:         started,
			FinishedAt:        finished,
			ProducedArtifacts: producedArtifacts(job),
			Err:               err,
		})
		if buildErr != nil {
			return JobResult{}, buildErr
		}
		mu.Lock()
		jobsRecorded[job.Name()] = jobResult
		mu.Unlock()
		return jobResult, nil
	}

	currentResult := func() *Result {
		mu.Lock()
		jobs := make([]JobResult, 0, len(jobOrder))
		for _, name := range jobOrder {
			if jobResult, ok := jobsRecorded[name]; ok {
				jobs = append(jobs, jobResult)
			}
		}
		mu.Unlock()

		result, buildErr := NewResult(ResultOptions{Groups: groupsRecorded, Jobs: jobs})
		if buildErr != nil {
			return &Result{}
		}
		return result
	}

	groups, err := e.scheduler.Schedule(ir)
	if err != nil {
		return nil, fmt.Errorf("schedule pipeline: %w", err)
	}
	for _, group := range groups {
		groupJobs := group.Jobs()
		groupResult, err := NewGroupResult(GroupResultOptions{
			Name:     group.Name(),
			JobCount: len(groupJobs),
		})
		if err != nil {
			return currentResult(), err
		}
		groupsRecorded = append(groupsRecorded, groupResult)
		if len(groupJobs) == 0 {
			continue
		}
		for i := range groupJobs {
			jobOrder = append(jobOrder, groupJobs[i].Name())
		}
		err = e.workers.Run(ctx, groupJobs, func(runCtx context.Context, job pipeline.Job) error {
			started := time.Now()
			startEvent := NewJobEvent(job, started)
			e.sink.JobStarted(startEvent)
			runErr := e.runner.Run(runCtx, job)
			finished := time.Now()

			status := JobStatusSucceeded
			if runErr != nil {
				status = JobStatusFailed
			}
			jobResult, recordErr := record(job, status, started, finished, runErr)
			if recordErr != nil {
				return recordErr
			}
			e.sink.JobFinished(NewJobEvent(job, finished), jobResult)
			if runErr != nil {
				return &ExecutionError{JobName: job.Name(), Err: runErr}
			}
			return nil
		})
		if err != nil {
			return currentResult(), err
		}
	}

	return currentResult(), nil
}

func producedArtifacts(job pipeline.Job) []pipeline.Artifact {
	artifact := job.OutputArtifact()
	if !artifact.Configured() {
		return nil
	}
	return []pipeline.Artifact{artifact}
}

type noopEventSink struct{}

func (noopEventSink) JobStarted(JobEvent)             {}
func (noopEventSink) JobFinished(JobEvent, JobResult) {}
