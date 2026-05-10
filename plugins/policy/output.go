package policy

import (
	"fmt"
	"io"

	log "github.com/caarlos0/log"

	"github.com/edelwud/terraci/plugins/internal/cliout"
	"github.com/edelwud/terraci/plugins/policy/internal/domain"
)

func outputResult(w io.Writer, format string, summary *domain.Summary, shouldBlock bool) error {
	if format == "json" {
		if err := cliout.WriteJSON(w, summary); err != nil {
			return err
		}
		return blockingError(summary, shouldBlock)
	}

	return outputText(summary, shouldBlock)
}

func blockingError(summary *domain.Summary, shouldBlock bool) error {
	if !shouldBlock {
		return nil
	}
	return fmt.Errorf("policy check failed with %d failures", summary.TotalFailures)
}

func outputText(summary *domain.Summary, shouldBlock bool) error {
	log.Info("summary")
	log.IncreasePadding()
	log.WithField("total", summary.TotalModules).Info("modules")
	if summary.PassedModules > 0 {
		log.WithField("count", summary.PassedModules).Info("passed")
	}
	if summary.WarnedModules > 0 {
		log.WithField("count", summary.WarnedModules).Warn("warned")
	}
	if summary.FailedModules > 0 {
		log.WithField("count", summary.FailedModules).Error("failed")
	}
	log.DecreasePadding()

	for _, result := range summary.Results {
		if result.Status() == "pass" {
			continue
		}
		log.WithField("module", result.Module).WithField("status", result.Status()).Info("module result")
		log.IncreasePadding()
		for _, failure := range result.Failures {
			log.WithField("namespace", failure.Namespace).WithField("message", failure.Message).Error("failure")
		}
		for _, warning := range result.Warnings {
			log.WithField("namespace", warning.Namespace).WithField("message", warning.Message).Warn("warning")
		}
		log.DecreasePadding()
	}

	if shouldBlock {
		log.Error("policy check FAILED")
		return blockingError(summary, shouldBlock)
	}

	if summary.HasWarnings() {
		log.Warn("policy check passed with warnings")
	} else {
		log.Info("policy check PASSED")
	}

	return nil
}
