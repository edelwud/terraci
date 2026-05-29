package execution

import (
	"errors"
	"fmt"
	"time"
)

// JobStatus is the status of an executed job.
type JobStatus string

const (
	JobStatusSucceeded JobStatus = "succeeded"
	JobStatusFailed    JobStatus = "failed"
)

func (s JobStatus) valid() bool {
	switch s {
	case JobStatusSucceeded, JobStatusFailed:
		return true
	default:
		return false
	}
}

// JobResultOptions describes one job execution result.
type JobResultOptions struct {
	Name        string
	Status      JobStatus
	StartedAt   time.Time
	FinishedAt  time.Time
	ArtifactIDs []string
	Err         error
}

// JobResult is the immutable execution outcome for one job.
type JobResult struct {
	name        string
	status      JobStatus
	startedAt   time.Time
	finishedAt  time.Time
	artifactIDs []string
	err         error
}

// NewJobResult validates and constructs a job result.
func NewJobResult(opts JobResultOptions) (JobResult, error) {
	if opts.Name == "" {
		return JobResult{}, errors.New("execution job result name is required")
	}
	if !opts.Status.valid() {
		return JobResult{}, fmt.Errorf("invalid execution job status %q", opts.Status)
	}
	return JobResult{
		name:        opts.Name,
		status:      opts.Status,
		startedAt:   opts.StartedAt,
		finishedAt:  opts.FinishedAt,
		artifactIDs: append([]string(nil), opts.ArtifactIDs...),
		err:         opts.Err,
	}, nil
}

func (r JobResult) Name() string            { return r.name }
func (r JobResult) Status() JobStatus       { return r.status }
func (r JobResult) StartedAt() time.Time    { return r.startedAt }
func (r JobResult) FinishedAt() time.Time   { return r.finishedAt }
func (r JobResult) Err() error              { return r.err }
func (r JobResult) Failed() bool            { return r.status == JobStatusFailed }
func (r JobResult) ArtifactIDs() []string   { return append([]string(nil), r.artifactIDs...) }
func (r JobResult) Duration() time.Duration { return r.finishedAt.Sub(r.startedAt) }
func (r JobResult) clone() JobResult {
	r.artifactIDs = append([]string(nil), r.artifactIDs...)
	return r
}

// GroupResultOptions describes one scheduled execution group result.
type GroupResultOptions struct {
	Name     string
	JobCount int
}

// GroupResult describes one scheduled execution group.
type GroupResult struct {
	name     string
	jobCount int
}

// NewGroupResult validates and constructs a group result.
func NewGroupResult(opts GroupResultOptions) (GroupResult, error) {
	if opts.Name == "" {
		return GroupResult{}, errors.New("execution group result name is required")
	}
	if opts.JobCount < 0 {
		return GroupResult{}, fmt.Errorf("execution group %q job count must be non-negative", opts.Name)
	}
	return GroupResult{name: opts.Name, jobCount: opts.JobCount}, nil
}

func (r GroupResult) Name() string  { return r.name }
func (r GroupResult) JobCount() int { return r.jobCount }

// ResultOptions describes an aggregate execution result.
type ResultOptions struct {
	Groups []GroupResult
	Jobs   []JobResult
}

// Result is the immutable aggregate outcome for a run.
type Result struct {
	groups []GroupResult
	jobs   []JobResult
}

// NewResult constructs an immutable aggregate execution result.
func NewResult(opts ResultOptions) (*Result, error) {
	return &Result{
		groups: append([]GroupResult(nil), opts.Groups...),
		jobs:   cloneJobResults(opts.Jobs),
	}, nil
}

// Groups returns defensive group result copies.
func (r *Result) Groups() []GroupResult {
	if r == nil || len(r.groups) == 0 {
		return nil
	}
	return append([]GroupResult(nil), r.groups...)
}

// Jobs returns defensive job result copies.
func (r *Result) Jobs() []JobResult {
	if r == nil || len(r.jobs) == 0 {
		return nil
	}
	return cloneJobResults(r.jobs)
}

// Clone returns a defensive result copy.
func (r *Result) Clone() *Result {
	if r == nil {
		return nil
	}
	cloned, err := NewResult(ResultOptions{Groups: r.groups, Jobs: r.jobs})
	if err != nil {
		return &Result{}
	}
	return cloned
}

// Failed returns the first failed job result, if any.
func (r *Result) Failed() (JobResult, bool) {
	if r == nil {
		return JobResult{}, false
	}
	for _, job := range r.jobs {
		if job.Failed() {
			return job.clone(), true
		}
	}
	return JobResult{}, false
}

// Stats returns aggregate execution counters and total job duration.
func (r *Result) Stats() Stats {
	if r == nil {
		return Stats{}
	}
	stats := Stats{groups: len(r.groups), jobs: len(r.jobs)}
	for _, job := range r.jobs {
		switch job.status {
		case JobStatusSucceeded:
			stats.succeeded++
		case JobStatusFailed:
			stats.failed++
		}
		stats.duration += job.Duration()
	}
	return stats
}

// Stats is an immutable aggregate execution summary.
type Stats struct {
	groups    int
	jobs      int
	succeeded int
	failed    int
	duration  time.Duration
}

func (s Stats) Groups() int             { return s.groups }
func (s Stats) Jobs() int               { return s.jobs }
func (s Stats) Succeeded() int          { return s.succeeded }
func (s Stats) Failed() int             { return s.failed }
func (s Stats) Duration() time.Duration { return s.duration }

func cloneJobResults(in []JobResult) []JobResult {
	if len(in) == 0 {
		return nil
	}
	out := make([]JobResult, len(in))
	for i := range in {
		out[i] = in[i].clone()
	}
	return out
}
