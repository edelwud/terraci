package summaryengine

import (
	"fmt"
	"html"
	"strings"

	"github.com/edelwud/terraci/pkg/ci"
)

func renderPlanRow(p *ci.PlanResult) string {
	status := statusIcon(p.Status)
	module := "`" + escapeInlineCode(p.ModuleID) + "`"
	summary := p.Summary
	if p.Error != "" {
		summary = "err: " + Truncate(p.Error, maxErrorLength)
	}
	if summary == "" {
		summary = "-"
	}
	return fmt.Sprintf("| %s | %s | %s |\n",
		escapeMarkdownTableCell(status),
		escapeMarkdownTableCell(module),
		escapeMarkdownTableCell(summary),
	)
}

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

func FormatMonthlyCost(cost float64) string {
	if cost == 0 {
		return "$0"
	}
	if cost < minCostThreshold {
		return "<$0.01"
	}
	if cost >= thousandCostThreshold {
		return fmt.Sprintf("$%.0f", cost)
	}
	if cost >= 1 {
		return fmt.Sprintf("$%.2f", cost)
	}
	return fmt.Sprintf("$%.4f", cost)
}

func FormatCostDiff(diff float64) string {
	if diff == 0 {
		return "$0"
	}
	if diff > 0 {
		if diff >= thousandCostThreshold {
			return fmt.Sprintf("+$%.0f", diff)
		}
		if diff >= 1 {
			return fmt.Sprintf("+$%.2f", diff)
		}
		return fmt.Sprintf("+$%.4f", diff)
	}
	diff = -diff
	if diff >= thousandCostThreshold {
		return fmt.Sprintf("-$%.0f", diff)
	}
	if diff >= 1 {
		return fmt.Sprintf("-$%.2f", diff)
	}
	return fmt.Sprintf("-$%.4f", diff)
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

func renderExpandableDetails(p *ci.PlanResult) string {
	var sb strings.Builder
	title := p.ModuleID
	if p.Summary != "" && p.Summary != noChangesSummary {
		title = fmt.Sprintf("%s (%s)", p.ModuleID, p.Summary)
	}
	if p.Status == ci.PlanStatusFailed {
		title = "FAILED " + p.ModuleID
	}

	fmt.Fprintf(&sb, "\n<details>\n<summary>%s</summary>\n\n", escapeHTMLText(title))
	if p.StructuredDetails != "" {
		sb.WriteString(p.StructuredDetails)
		sb.WriteString("\n\n")
	}
	if p.RawPlanOutput != "" {
		sb.WriteString("<details>\n<summary>Full plan output</summary>\n\n")
		sb.WriteString("```diff\n")
		sb.WriteString(Truncate(p.RawPlanOutput, maxDetailsLength))
		sb.WriteString("\n```\n")
		sb.WriteString("</details>\n")
	}
	sb.WriteString("</details>\n")
	return sb.String()
}

func escapeMarkdownTableCell(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "|", "\\|")
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return strings.ReplaceAll(s, "\n", "<br>")
}

func escapeMarkdownText(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return strings.ReplaceAll(s, "\n", " ")
}

func escapeInlineCode(s string) string {
	return strings.ReplaceAll(s, "`", "\\`")
}

func escapeHTMLText(s string) string {
	return html.EscapeString(escapeMarkdownText(s))
}

func Truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
