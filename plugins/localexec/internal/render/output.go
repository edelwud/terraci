package render

import (
	"fmt"
	"time"

	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/execution"
	"github.com/edelwud/terraci/pkg/log"
	"github.com/edelwud/terraci/pkg/pipeline"
)

type Output interface {
	Completed(result *execution.Result, summaryReport *ci.Report)
	Failure(result *execution.Result, execErr error) error
}

type LogOutput struct{}

func NewLogOutput() Output {
	return LogOutput{}
}

func (o LogOutput) Completed(result *execution.Result, summaryReport *ci.Report) {
	var succeeded, failed int
	var totalDuration time.Duration
	if result == nil {
		result = &execution.Result{}
	}

	for _, job := range result.Jobs {
		switch job.Status {
		case execution.JobStatusSucceeded:
			succeeded++
		case execution.JobStatusFailed:
			failed++
		case execution.JobStatusSkipped:
		}
		totalDuration += job.FinishedAt.Sub(job.StartedAt)
	}

	if len(result.Groups) > 0 {
		log.Info("")
		log.Info("── stages ───────────────────────────")
		for _, group := range result.Groups {
			log.WithField("jobs", group.JobCount).Info(group.Name)
		}
		log.Info("─────────────────────────────────────")
	}

	if len(result.Jobs) > 0 {
		log.Info("")
		log.Info("── summary ──────────────────────────")
		for _, job := range result.Jobs {
			duration := job.FinishedAt.Sub(job.StartedAt).Truncate(time.Millisecond)
			entry := log.WithField("status", string(job.Status)).WithField("duration", duration)
			if job.Err != nil {
				entry.WithError(job.Err).Warn(job.Name)
			} else {
				entry.Info(job.Name)
			}
		}
		log.Info("─────────────────────────────────────")
	}

	entry := log.WithField("stages", len(result.Groups)).
		WithField("jobs", len(result.Jobs)).
		WithField("succeeded", succeeded).
		WithField("failed", failed).
		WithField("duration", totalDuration.Truncate(time.Millisecond))
	entry.Info("local execution completed")

	if summaryReport == nil {
		return
	}

	log.Info("")
	rendered := SummaryReportCLI(summaryReport)
	if rendered != "" {
		fmt.Println(rendered)
	}
}

func (LogOutput) Failure(result *execution.Result, execErr error) error {
	if result != nil {
		if failed := result.Failed(); failed != nil {
			return fmt.Errorf("local-exec failed in job %s: %w", failed.Name, failed.Err)
		}
	}
	return execErr
}

type ProgressReporter struct{}

func NewProgressReporter() execution.EventSink {
	return ProgressReporter{}
}

func (ProgressReporter) JobStarted(job *pipeline.Job) {
	entry := log.WithField("job", job.Name).WithField("stage", job.Phase.String())
	if job.Module != nil {
		entry = entry.WithField("module", job.Module.ID())
	}
	entry.Info("job started")
}

func (ProgressReporter) JobFinished(job *pipeline.Job, result *execution.JobResult) {
	if result == nil {
		return
	}
	entry := log.WithField("job", job.Name).
		WithField("stage", job.Phase.String()).
		WithField("status", result.Status)
	if job.Module != nil {
		entry = entry.WithField("module", job.Module.ID())
	}
	if result.Err != nil {
		entry.WithError(result.Err).Warn("job finished")
		return
	}
	entry.Info("job finished")
}
