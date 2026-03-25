package policy

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/log"
	"github.com/edelwud/terraci/pkg/plugin"
	policyengine "github.com/edelwud/terraci/plugins/policy/internal"
)

// ContributeToSummary loads policy results and contributes to summary comment.
func (p *Plugin) ContributeToSummary(_ context.Context, appCtx *plugin.AppContext, execCtx *plugin.ExecutionContext) error {
	// Try common locations for policy results
	paths := []string{
		filepath.Join(".terraci", "policy-results.json"),
		"policy-results.json",
		filepath.Join(appCtx.WorkDir, ".terraci", "policy-results.json"),
	}

	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		var raw policyengine.Summary
		if err := json.Unmarshal(data, &raw); err != nil {
			log.WithField("path", path).WithError(err).Debug("failed to parse policy results")
			continue
		}

		// Convert to CI types and store in execution context
		policySummary := toCIPolicySummary(&raw)
		execCtx.SetData("policy:summary", policySummary)

		log.WithField("modules", policySummary.TotalModules).
			WithField("failures", policySummary.TotalFailures).
			WithField("warnings", policySummary.TotalWarnings).
			Info("loaded policy results")

		return nil
	}

	return nil
}

func toCIPolicySummary(s *policyengine.Summary) *ci.PolicySummary {
	results := make([]ci.PolicyResult, len(s.Results))
	for i, r := range s.Results {
		failures := make([]ci.PolicyViolation, len(r.Failures))
		for j, f := range r.Failures {
			failures[j] = ci.PolicyViolation{Namespace: f.Namespace, Message: f.Message}
		}
		warnings := make([]ci.PolicyViolation, len(r.Warnings))
		for j, w := range r.Warnings {
			warnings[j] = ci.PolicyViolation{Namespace: w.Namespace, Message: w.Message}
		}
		results[i] = ci.PolicyResult{
			Module:   r.Module,
			Failures: failures,
			Warnings: warnings,
		}
	}
	return &ci.PolicySummary{
		TotalModules:  s.TotalModules,
		PassedModules: s.PassedModules,
		WarnedModules: s.WarnedModules,
		FailedModules: s.FailedModules,
		TotalFailures: s.TotalFailures,
		TotalWarnings: s.TotalWarnings,
		Results:       results,
	}
}
