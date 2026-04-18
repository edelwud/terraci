package internal

import (
	"fmt"
	"time"

	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/execution"
	"github.com/edelwud/terraci/pkg/log"
	"github.com/edelwud/terraci/pkg/pipeline"
)

type executionOutput interface {
	DryRun(plan *execution.Plan) error
	Completed(result *execution.Result)
	Failure(result *execution.Result, execErr error) error
}

type logOutput struct {
	serviceDir string
}

func (logOutput) DryRun(plan *execution.Plan) error {
	log.Info("local-exec dry run")
	for _, group := range defaultDryRunGroups(plan) {
		for _, job := range group.Jobs {
			log.WithField("group", group.Name).WithField("job", job.Name).Info("scheduled job")
		}
	}
	return nil
}

func (o logOutput) Completed(result *execution.Result) {
	if result == nil || len(result.Jobs) == 0 {
		log.Info("local execution completed (no jobs)")
		return
	}

	var succeeded, failed int
	var totalDuration time.Duration
	for _, job := range result.Jobs {
		switch job.Status {
		case execution.JobStatusSucceeded:
			succeeded++
		case execution.JobStatusFailed:
			failed++
		case execution.JobStatusSkipped:
			// not counted
		}
		totalDuration += job.FinishedAt.Sub(job.StartedAt)
	}

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
	log.WithField("succeeded", succeeded).
		WithField("failed", failed).
		WithField("total", len(result.Jobs)).
		WithField("duration", totalDuration.Truncate(time.Millisecond)).
		Info("local execution completed")

	o.printReports()
}

func (o logOutput) printReports() {
	if o.serviceDir == "" {
		return
	}
	reports, err := ci.LoadReports(o.serviceDir)
	if err != nil || len(reports) == 0 {
		return
	}

	log.Info("")
	log.Info("── reports ──────────────────────────")
	for _, r := range reports {
		entry := log.WithField("status", string(r.Status))
		if r.Summary != "" {
			entry = entry.WithField("summary", r.Summary)
		}
		switch r.Status {
		case ci.ReportStatusFail:
			entry.Warn(r.Title)
		case ci.ReportStatusPass, ci.ReportStatusWarn:
			entry.Info(r.Title)
		}
	}
	log.Info("─────────────────────────────────────")
}

func (logOutput) Failure(result *execution.Result, execErr error) error {
	if result != nil {
		if failed := result.Failed(); failed != nil {
			return fmt.Errorf("local-exec failed in job %s: %w", failed.Name, failed.Err)
		}
	}
	return execErr
}

type progressReporter struct{}

func (progressReporter) JobStarted(job *pipeline.Job) {
	log.WithField("job", job.Name).Info("job started")
}

func (progressReporter) JobFinished(job *pipeline.Job, result *execution.JobResult) {
	if result == nil {
		return
	}
	entry := log.WithField("job", job.Name).WithField("status", result.Status)
	if result.Err != nil {
		entry.WithError(result.Err).Warn("job finished")
		return
	}
	entry.Info("job finished")
}

func defaultDryRunGroups(plan *execution.Plan) []execution.JobGroup {
	return execution.DefaultScheduler{}.Schedule(plan)
}
