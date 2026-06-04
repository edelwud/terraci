package reportrender

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/edelwud/terraci/pkg/ci"
)

// CLIReport renders a report for terminal output.
func CLIReport(report *ci.Report) (string, error) {
	if report == nil {
		return "", nil
	}

	var sb strings.Builder
	sb.WriteString(renderSectionTitle(report.Title()))
	if report.Summary() != "" {
		sb.WriteString("\n")
		sb.WriteString(report.Summary())
	}

	for i, section := range report.Sections() {
		rendered, err := CLISection(section)
		if err != nil {
			return "", fmt.Errorf("render report section %d: %w", i, err)
		}
		if rendered == "" {
			continue
		}
		sb.WriteString("\n\n")
		sb.WriteString(rendered)
	}

	return strings.TrimSpace(sb.String()), nil
}

// CLISection renders one render-ready section for terminal output.
func CLISection(section ci.ReportSection) (string, error) {
	rendered, err := ci.DecodeRenderSection(section)
	if err != nil {
		return "", err
	}
	return renderCLIRenderSection(section, rendered), nil
}

func renderCLIRenderSection(section ci.ReportSection, rendered ci.RenderSection) string {
	var sb strings.Builder
	sb.WriteString(renderCLISectionHeader(section))
	blocks := rendered.Blocks()
	for i := range blocks {
		part := renderCLIBlock(blocks[i])
		if part == "" {
			continue
		}
		sb.WriteString("\n\n")
		sb.WriteString(part)
	}
	return strings.TrimSpace(sb.String())
}

func renderCLIBlock(block ci.RenderBlock) string {
	switch block.Kind() {
	case ci.RenderBlockKindText:
		return renderCLITextBlock(block)
	case ci.RenderBlockKindList:
		return renderCLIListBlock(block)
	case ci.RenderBlockKindTable:
		return renderCLITableBlock(block)
	case ci.RenderBlockKindDetails:
		return renderCLIDetailsBlock(block)
	default:
		return ""
	}
}

func renderCLITextBlock(block ci.RenderBlock) string {
	var sb strings.Builder
	if block.Title() != "" {
		sb.WriteString(renderSubsectionTitle(block.Title()))
		sb.WriteString("\n")
	}
	sb.WriteString(formatPlainValue(block.Text()))
	return sb.String()
}

func renderCLIListBlock(block ci.RenderBlock) string {
	var sb strings.Builder
	if block.Title() != "" {
		sb.WriteString(renderSubsectionTitle(block.Title()))
		sb.WriteString("\n")
	}
	for _, item := range block.Items() {
		fmt.Fprintf(&sb, "• %s\n", formatPlainValue(item))
	}
	return strings.TrimSpace(sb.String())
}

func renderCLITableBlock(block ci.RenderBlock) string {
	table := block.Table()
	columns := table.Columns()
	tableRows := table.Rows()
	rows := make([][]string, 0, len(tableRows)+1)
	header := make([]string, 0, len(columns))
	for _, column := range columns {
		header = append(header, formatPlainValue(column.Title()))
	}
	rows = append(rows, header)
	for _, row := range tableRows {
		cells := row.Cells()
		rendered := make([]string, len(columns))
		for idx := range columns {
			if idx < len(cells) {
				rendered[idx] = formatPlainValue(cells[idx])
			}
		}
		rows = append(rows, rendered)
	}
	var sb strings.Builder
	if block.Title() != "" {
		sb.WriteString(renderSubsectionTitle(block.Title()))
		sb.WriteString("\n")
	}
	sb.WriteString(renderTable(rows))
	return sb.String()
}

func renderCLIDetailsBlock(block ci.RenderBlock) string {
	details := block.Details()
	var sb strings.Builder
	sb.WriteString(renderSubsectionTitle(details.Summary()))
	if details.Body() != "" {
		sb.WriteString("\n")
		sb.WriteString(indentBlock(cleanMarkdown(details.Body()), "  "))
	}
	return sb.String()
}

func renderCLISectionHeader(section ci.ReportSection) string {
	var sb strings.Builder
	sb.WriteString(renderSubsectionTitle(sectionTitle(section)))
	if section.Summary() != "" {
		sb.WriteString("\n")
		fmt.Fprintf(&sb, "%s - %s", StatusLabel(section.Status()), section.Summary())
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

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
