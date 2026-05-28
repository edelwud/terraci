package policy

import (
	"errors"
	"fmt"
	"io"

	log "github.com/caarlos0/log"

	"github.com/edelwud/terraci/pkg/plugin/cliout"
	policyengine "github.com/edelwud/terraci/plugins/policy/internal"
)

func outputResult(w io.Writer, format cliout.Format, summary *policyengine.Summary, shouldBlock bool) error {
	if summary == nil {
		return errors.New("policy summary is nil")
	}

	switch format {
	case cliout.FormatJSON:
		if err := cliout.WriteJSON(w, summary); err != nil {
			return err
		}
		return blockingError(summary, shouldBlock)
	case cliout.FormatText, "":
		return outputText(summary, shouldBlock)
	default:
		return fmt.Errorf("unsupported output format %q: must be one of: text, json", format)
	}
}

func blockingError(summary *policyengine.Summary, shouldBlock bool) error {
	if summary == nil {
		return errors.New("policy summary is nil")
	}
	if !shouldBlock {
		return nil
	}
	return fmt.Errorf("policy check failed with %d failures", summary.TotalFailures)
}

func outputText(summary *policyengine.Summary, shouldBlock bool) error {
	if summary == nil {
		return errors.New("policy summary is nil")
	}

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
		if result.Status() == policyengine.StatusPass {
			continue
		}
		log.WithField("module", result.Module).WithField("status", result.Status().String()).Info("module result")
		log.IncreasePadding()
		for _, failure := range result.Failures {
			log.WithField("namespace", failure.Namespace.String()).WithField("message", failure.Message).Error("failure")
		}
		for _, warning := range result.Warnings {
			log.WithField("namespace", warning.Namespace.String()).WithField("message", warning.Message).Warn("warning")
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
