package policy

import (
	"errors"
	"fmt"

	"github.com/edelwud/terraci/pkg/ci"
	policyengine "github.com/edelwud/terraci/plugins/policy/internal"
)

type policyReportRequest struct {
	Summary  *policyengine.Summary
	Artifact ci.ArtifactContext
}

func buildPolicyReport(req policyReportRequest) (*ci.Report, error) {
	summary := req.Summary
	if summary == nil {
		return nil, errors.New("policy summary is nil")
	}

	status := ci.StatusFromCounts(summary.FailedModules, summary.WarnedModules)

	rows := make([][]string, 0, len(summary.Results))
	for i := range summary.Results {
		result := &summary.Results[i]
		for _, failure := range result.Failures {
			rows = append(rows, []string{result.Module, "fail", failure.Namespace, failure.Message})
		}
		for _, warning := range result.Warnings {
			rows = append(rows, []string{result.Module, "warn", warning.Namespace, warning.Message})
		}
	}

	summaryText := fmt.Sprintf("%d modules: %d passed, %d warned, %d failed",
		summary.TotalModules, summary.PassedModules, summary.WarnedModules, summary.FailedModules)
	blocks := make([]ci.RenderBlock, 0, 1)
	if len(rows) > 0 {
		blocks = append(blocks, ci.RenderTableBlock("", []string{"Module", "Severity", "Namespace", "Message"}, rows))
	}
	report, err := ci.NewRenderedReport(ci.RenderedReportOptions{
		Producer: pluginName,
		Title:    "Policy Check",
		Status:   status,
		Summary:  summaryText,
		Artifact: req.Artifact,
		Sections: []ci.RenderedSectionOptions{{
			Title:   "Policy Check",
			Summary: summaryText,
			Blocks:  blocks,
		}},
	})
	if err != nil {
		return nil, fmt.Errorf("build policy report: %w", err)
	}

	return report, nil
}
