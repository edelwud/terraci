package policy

import (
	"encoding/json"
	"fmt"
	"io"

	log "github.com/caarlos0/log"

	policyengine "github.com/edelwud/terraci/plugins/policy/internal"
)

func outputResult(w io.Writer, format string, summary *policyengine.Summary, shouldBlock bool) error {
	if format == "json" {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(summary)
	}

	return outputText(summary, shouldBlock)
}

func outputText(summary *policyengine.Summary, shouldBlock bool) error {
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
		return fmt.Errorf("policy check failed with %d failures", summary.TotalFailures)
	}

	if summary.HasWarnings() {
		log.Warn("policy check passed with warnings")
	} else {
		log.Info("policy check PASSED")
	}

	return nil
}
