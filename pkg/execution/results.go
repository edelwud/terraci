package execution

import "time"

// JobStatus is the status of an executed job.
type JobStatus string

const (
	JobStatusSucceeded JobStatus = "succeeded"
	JobStatusFailed    JobStatus = "failed"
	JobStatusSkipped   JobStatus = "skipped"
)

// JobResult is the execution outcome for one job.
type JobResult struct {
	Name        string
	Status      JobStatus
	StartedAt   time.Time
	FinishedAt  time.Time
	ArtifactIDs []string
	Err         error
}

// Result is the aggregate outcome for a run.
type Result struct {
	Jobs []*JobResult
}

// Failed returns the first failed job result, if any.
func (r *Result) Failed() *JobResult {
	if r == nil {
		return nil
	}
	for _, job := range r.Jobs {
		if job.Status == JobStatusFailed {
			return job
		}
	}
	return nil
}
