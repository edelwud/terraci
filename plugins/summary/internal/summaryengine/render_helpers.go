package summaryengine

import (
	"fmt"
	"strings"

	"github.com/edelwud/terraci/pkg/ci"
)

func planSummary(p ci.PlanResult) string {
	if p.Error != "" {
		return "err: " + Truncate(p.Error, maxErrorLength)
	}
	if p.Summary != "" {
		return p.Summary
	}
	return "-"
}

func planDetailsTitle(p ci.PlanResult) string {
	title := p.ModuleID
	if p.Summary != "" && p.Summary != noChangesSummary {
		title = fmt.Sprintf("%s (%s)", p.ModuleID, p.Summary)
	}
	if p.Status == ci.PlanStatusFailed {
		title = "FAILED " + p.ModuleID
	}
	return title
}

func planDetailsBody(p ci.PlanResult) string {
	var sb strings.Builder
	if p.StructuredDetails != "" {
		sb.WriteString(p.StructuredDetails)
	}
	if p.RawPlanOutput != "" {
		if sb.Len() > 0 {
			sb.WriteString("\n\n")
		}
		sb.WriteString("#### Full plan output\n\n")
		sb.WriteString("```diff\n")
		sb.WriteString(Truncate(p.RawPlanOutput, maxDetailsLength))
		sb.WriteString("\n```")
	}
	return sb.String()
}

func statusIcon(status ci.PlanStatus) string {
	switch status {
	case ci.PlanStatusSuccess, ci.PlanStatusNoChanges:
		return "ok"
	case ci.PlanStatusChanges:
		return "changes"
	case ci.PlanStatusFailed:
		return "failed"
	case ci.PlanStatusPending:
		return "pending"
	case ci.PlanStatusRunning:
		return "running"
	default:
		return "?"
	}
}

func planStatusTone(status ci.PlanStatus) ci.RenderTone {
	switch status {
	case ci.PlanStatusSuccess, ci.PlanStatusNoChanges:
		return ci.RenderToneSuccess
	case ci.PlanStatusChanges, ci.PlanStatusPending, ci.PlanStatusRunning:
		return ci.RenderToneWarning
	case ci.PlanStatusFailed:
		return ci.RenderToneFailure
	default:
		return ci.RenderToneNeutral
	}
}

func Truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
