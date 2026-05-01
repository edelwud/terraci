package render

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/edelwud/terraci/pkg/ci"
)

func SummaryReportCLI(report *ci.Report) string {
	if report == nil {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(renderSectionTitle(report.Title))
	if report.Summary != "" {
		sb.WriteString("\n")
		sb.WriteString(report.Summary)
	}

	for _, section := range report.Sections {
		rendered := renderCLISection(section)
		if rendered == "" {
			continue
		}
		sb.WriteString("\n\n")
		sb.WriteString(rendered)
	}

	return strings.TrimSpace(sb.String())
}

func renderCLISection(section ci.ReportSection) string {
	switch section.Kind {
	case ci.ReportSectionKindOverview:
		return renderCLIOverviewSection(section)
	case ci.ReportSectionKindModuleTable:
		return renderCLIModuleTableSection(section)
	case costChangesSectionKind:
		return renderCLICostChangesSection(section)
	case ci.ReportSectionKindFindings:
		return renderCLIFindingsSection(section)
	case ci.ReportSectionKindDependencyUpdates:
		return renderCLIDependencyUpdatesSection(section)
	default:
		return ""
	}
}

func renderCLIOverviewSection(section ci.ReportSection) string {
	if section.Overview == nil {
		return ""
	}

	stats := section.Overview.PlanStats
	var parts []string
	if stats.Changes > 0 {
		parts = append(parts, fmt.Sprintf("%d with changes", stats.Changes))
	}
	if stats.NoChanges > 0 {
		parts = append(parts, fmt.Sprintf("%d no changes", stats.NoChanges))
	}
	if stats.Failed > 0 {
		parts = append(parts, fmt.Sprintf("%d failed", stats.Failed))
	}
	if stats.Pending > 0 {
		parts = append(parts, fmt.Sprintf("%d pending", stats.Pending))
	}
	if stats.Running > 0 {
		parts = append(parts, fmt.Sprintf("%d running", stats.Running))
	}

	var sb strings.Builder
	sb.WriteString(renderSubsectionTitle(section.Title))
	sb.WriteString("\n")
	if len(parts) == 0 {
		fmt.Fprintf(&sb, "%d modules analyzed", stats.Total)
	} else {
		fmt.Fprintf(&sb, "%d modules: %s", stats.Total, strings.Join(parts, " | "))
	}

	for _, overview := range section.Overview.Reports {
		sb.WriteString("\n")
		fmt.Fprintf(&sb, "• %s %s", reportStatusLabel(overview.Status), overview.Title)
		if overview.Summary != "" {
			sb.WriteString(": ")
			sb.WriteString(overview.Summary)
		}
	}

	return sb.String()
}

func renderCLIModuleTableSection(section ci.ReportSection) string {
	if section.ModuleTable == nil {
		return ""
	}

	rows := make([][]string, 0, len(section.ModuleTable.Rows)+1)
	rows = append(rows, []string{"Status", "Module", "Summary"})
	for i := range section.ModuleTable.Rows {
		row := section.ModuleTable.Rows[i]
		record := []string{
			planStatusLabel(row.Status),
			displayValue(row.ModuleID),
			displayPlanSummary(row),
		}
		rows = append(rows, record)
	}

	var sb strings.Builder
	sb.WriteString(renderSubsectionTitle(section.Title))
	sb.WriteString("\n")
	sb.WriteString(renderTable(rows))

	for i := range section.ModuleTable.Rows {
		row := section.ModuleTable.Rows[i]
		if row.StructuredDetails == "" && row.RawPlanOutput == "" && row.Error == "" {
			continue
		}
		sb.WriteString("\n\n")
		title := row.ModuleID
		if row.Status == ci.PlanStatusFailed {
			title = "FAILED " + row.ModuleID
		} else if row.Summary != "" {
			title = row.ModuleID + " (" + row.Summary + ")"
		}
		sb.WriteString(renderSubsectionTitle(title))
		if row.StructuredDetails != "" {
			sb.WriteString("\n")
			sb.WriteString(indentBlock(cleanMarkdown(row.StructuredDetails), "  "))
		}
		if row.RawPlanOutput != "" {
			sb.WriteString("\n")
			sb.WriteString(renderSubsectionTitle("Full plan output"))
			sb.WriteString("\n")
			sb.WriteString(indentBlock(row.RawPlanOutput, "    "))
		}
	}

	return sb.String()
}

func renderCLICostChangesSection(section ci.ReportSection) string {
	payload, ok := decodeCostChangesPayload(section)
	if !ok {
		return ""
	}

	rows := [][]string{{"Module", "Before", "After", "Diff", "Notes"}}
	for _, row := range payload.Rows {
		rows = append(rows, []string{
			row.ModulePath,
			formatMonthlyCost(row.Before),
			formatMonthlyCost(row.After),
			formatCostDiff(row.Diff),
			firstNonEmpty(row.Error, row.Notes, "-"),
		})
	}

	var sb strings.Builder
	sb.WriteString(renderSectionHeader(section))
	sb.WriteString("\n")
	sb.WriteString(renderTable(rows))
	sb.WriteString("\n")
	fmt.Fprintf(&sb, "Totals: %s %s -> %s", formatMonthlyCost(payload.Totals.Before), formatCostDiff(payload.Totals.Diff), formatMonthlyCost(payload.Totals.After))
	return sb.String()
}

func renderCLIFindingsSection(section ci.ReportSection) string {
	if section.Findings == nil {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(renderSectionHeader(section))
	for _, row := range section.Findings.Rows {
		sb.WriteString("\n")
		sb.WriteString(renderSubsectionTitle(fmt.Sprintf("%s (%s)", row.ModulePath, row.Status)))
		for _, finding := range row.Findings {
			sb.WriteString("\n")
			sb.WriteString("• ")
			sb.WriteString(string(finding.Severity))
			sb.WriteString(": ")
			sb.WriteString(finding.Message)
			if finding.Namespace != "" {
				sb.WriteString(" (")
				sb.WriteString(finding.Namespace)
				sb.WriteString(")")
			}
		}
		sb.WriteString("\n")
	}

	return strings.TrimSpace(sb.String())
}

func renderCLIDependencyUpdatesSection(section ci.ReportSection) string {
	if section.DependencyUpdates == nil {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(renderSectionHeader(section))

	providers, modules := splitDependencyUpdateRows(section.DependencyUpdates.Rows)
	if len(providers) > 0 {
		sb.WriteString("\n")
		sb.WriteString(renderSubsectionTitle("Providers"))
		sb.WriteString("\n")
		rows := [][]string{{"Module", "Provider", "Current", "Latest", "Status"}}
		for i := range providers {
			row := providers[i]
			rows = append(rows, []string{row.ModulePath, row.Name, displayValue(row.Current), displayValue(row.Latest), dependencyUpdateStatusLabel(row.Status, row.Issue)})
		}
		sb.WriteString(renderTable(rows))
	}

	if len(modules) > 0 {
		sb.WriteString("\n")
		sb.WriteString(renderSubsectionTitle("Modules"))
		sb.WriteString("\n")
		rows := [][]string{{"Module", "Source", "Current", "Latest", "Status"}}
		for i := range modules {
			row := modules[i]
			rows = append(rows, []string{row.ModulePath, row.Name, displayValue(row.Current), displayValue(row.Latest), dependencyUpdateStatusLabel(row.Status, row.Issue)})
		}
		sb.WriteString(renderTable(rows))
	}

	return strings.TrimSpace(sb.String())
}

func renderSectionHeader(section ci.ReportSection) string {
	var sb strings.Builder
	sb.WriteString(renderSubsectionTitle(section.Title))
	if section.SectionSummary != "" {
		sb.WriteString("\n")
		fmt.Fprintf(&sb, "%s — %s", reportStatusLabel(section.Status), section.SectionSummary)
	}
	return sb.String()
}

func renderSectionTitle(title string) string {
	title = strings.TrimSpace(title)
	if title == "" {
		return ""
	}
	return title + "\n" + strings.Repeat("═", maxInt(8, lipgloss.Width(title)))
}

func renderSubsectionTitle(title string) string {
	title = strings.TrimSpace(title)
	if title == "" {
		return ""
	}
	return title + "\n" + strings.Repeat("─", maxInt(6, lipgloss.Width(title)))
}

func renderTable(rows [][]string) string {
	if len(rows) == 0 {
		return ""
	}

	widths := make([]int, len(rows[0]))
	for _, row := range rows {
		for idx, cell := range row {
			if idx >= len(widths) {
				continue
			}
			widths[idx] = maxInt(widths[idx], lipgloss.Width(cell))
		}
	}

	var sb strings.Builder
	sb.WriteString(renderTableBorder("┌", "┬", "┐", widths))
	sb.WriteString("\n")
	for rowIdx, row := range rows {
		sb.WriteString("│")
		for colIdx, width := range widths {
			cell := ""
			if colIdx < len(row) {
				cell = row[colIdx]
			}
			sb.WriteString(" ")
			sb.WriteString(padRight(cell, width))
			sb.WriteString(" │")
		}
		sb.WriteString("\n")
		if rowIdx == 0 {
			sb.WriteString(renderTableBorder("├", "┼", "┤", widths))
			sb.WriteString("\n")
		}
	}
	sb.WriteString(renderTableBorder("└", "┴", "┘", widths))
	return sb.String()
}

func renderTableBorder(left, mid, right string, widths []int) string {
	var sb strings.Builder
	sb.WriteString(left)
	for idx, width := range widths {
		sb.WriteString(strings.Repeat("─", width+2))
		if idx < len(widths)-1 {
			sb.WriteString(mid)
		}
	}
	sb.WriteString(right)
	return sb.String()
}

func padRight(s string, width int) string {
	padding := width - lipgloss.Width(s)
	if padding <= 0 {
		return s
	}
	return s + strings.Repeat(" ", padding)
}

func indentBlock(s, indent string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	for i := range lines {
		lines[i] = indent + lines[i]
	}
	return strings.Join(lines, "\n")
}

func cleanMarkdown(s string) string {
	s = strings.ReplaceAll(s, "**", "")
	s = strings.ReplaceAll(s, "`", "")
	s = strings.ReplaceAll(s, "<details>", "")
	s = strings.ReplaceAll(s, "</details>", "")
	s = strings.ReplaceAll(s, "<summary>", "")
	s = strings.ReplaceAll(s, "</summary>", "")
	s = strings.ReplaceAll(s, "```diff", "")
	s = strings.ReplaceAll(s, "```", "")
	return strings.TrimSpace(s)
}

func displayPlanSummary(row ci.ModuleTableRow) string {
	if row.Error != "" {
		return "err: " + row.Error
	}
	return firstNonEmpty(row.Summary, "-")
}

func formatMonthlyCost(cost float64) string {
	if cost == 0 {
		return "$0"
	}
	if cost < 0.01 {
		return "<$0.01"
	}
	if cost >= 1000 {
		return fmt.Sprintf("$%.0f", cost)
	}
	if cost >= 1 {
		return fmt.Sprintf("$%.2f", cost)
	}
	return fmt.Sprintf("$%.4f", cost)
}

func formatCostDiff(diff float64) string {
	if diff == 0 {
		return "$0"
	}
	if diff > 0 {
		if diff >= 1000 {
			return fmt.Sprintf("+$%.0f", diff)
		}
		if diff >= 1 {
			return fmt.Sprintf("+$%.2f", diff)
		}
		return fmt.Sprintf("+$%.4f", diff)
	}
	diff = -diff
	if diff >= 1000 {
		return fmt.Sprintf("-$%.0f", diff)
	}
	if diff >= 1 {
		return fmt.Sprintf("-$%.2f", diff)
	}
	return fmt.Sprintf("-$%.4f", diff)
}

func planStatusLabel(status ci.PlanStatus) string {
	return string(status)
}

func reportStatusLabel(status ci.ReportStatus) string {
	return string(status)
}

func splitDependencyUpdateRows(rows []ci.DependencyUpdateRow) (providers, modules []ci.DependencyUpdateRow) {
	providers = make([]ci.DependencyUpdateRow, 0, len(rows))
	modules = make([]ci.DependencyUpdateRow, 0, len(rows))
	for i := range rows {
		row := rows[i]
		switch row.Kind {
		case ci.DependencyKindProvider:
			providers = append(providers, row)
		case ci.DependencyKindModule:
			modules = append(modules, row)
		}
	}
	return providers, modules
}

func dependencyUpdateStatusLabel(status ci.DependencyUpdateStatus, issue string) string {
	switch status {
	case ci.DependencyUpdateStatusUpdateAvailable:
		return "update available"
	case ci.DependencyUpdateStatusApplied:
		return "applied"
	case ci.DependencyUpdateStatusSkipped:
		return "skipped: " + issue
	case ci.DependencyUpdateStatusError:
		return "error: " + issue
	case ci.DependencyUpdateStatusUpToDate:
		return "up to date"
	default:
		return "up to date"
	}
}

func displayValue(v string) string {
	if v == "" {
		return "-"
	}
	return v
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
