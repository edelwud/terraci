package render

import (
	"fmt"
	"time"

	log "github.com/caarlos0/log"

	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/execution"
)

type Output interface {
	Completed(result *execution.Result, summaryReport *ci.Report) error
	Failure(result *execution.Result, execErr error) error
}

type LogOutput struct{}

func NewLogOutput() Output {
	return LogOutput{}
}

func (o LogOutput) Completed(result *execution.Result, summaryReport *ci.Report) error {
	stats := result.Stats()
	groups := result.Groups()
	jobs := result.Jobs()

	if len(groups) > 0 {
		log.Info("")
		log.Info("── stages ───────────────────────────")
		for _, group := range groups {
			log.WithField("jobs", group.JobCount()).Info(group.Name())
		}
		log.Info("─────────────────────────────────────")
	}

	if len(jobs) > 0 {
		log.Info("")
		log.Info("── summary ──────────────────────────")
		for _, job := range jobs {
			duration := job.Duration().Truncate(time.Millisecond)
			entry := log.WithField("status", string(job.Status())).WithField("duration", duration)
			if job.Err() != nil {
				entry.WithError(job.Err()).Warn(job.Name())
			} else {
				entry.Info(job.Name())
			}
		}
		log.Info("─────────────────────────────────────")
	}

	entry := log.WithField("stages", stats.Groups()).
		WithField("jobs", stats.Jobs()).
		WithField("succeeded", stats.Succeeded()).
		WithField("failed", stats.Failed()).
		WithField("duration", stats.Duration().Truncate(time.Millisecond))
	entry.Info("local execution completed")

	if summaryReport == nil {
		return nil
	}

	log.Info("")
	rendered, err := SummaryReportCLI(summaryReport)
	if err != nil {
		return fmt.Errorf("render summary report: %w", err)
	}
	if rendered != "" {
		fmt.Println(rendered)
	}
	return nil
}

func (LogOutput) Failure(result *execution.Result, execErr error) error {
	if result != nil {
		if failed, ok := result.Failed(); ok {
			return fmt.Errorf("local-exec failed in job %s: %w", failed.Name(), failed.Err())
		}
	}
	return execErr
}

type ProgressReporter struct{}

func NewProgressReporter() execution.EventSink {
	return ProgressReporter{}
}

func (ProgressReporter) JobStarted(event execution.JobEvent) {
	entry := log.WithField("job", event.Name()).WithField("operation", event.Operation())
	if module := event.ModuleID(); module != "" {
		entry = entry.WithField("module", module)
	}
	entry.Info("job started")
}

func (ProgressReporter) JobFinished(event execution.JobEvent, result execution.JobResult) {
	entry := log.WithField("job", event.Name()).
		WithField("operation", event.Operation()).
		WithField("status", result.Status())
	if module := event.ModuleID(); module != "" {
		entry = entry.WithField("module", module)
	}
	if result.Err() != nil {
		entry.WithError(result.Err()).Warn("job finished")
		return
	}
	entry.Info("job finished")
}
