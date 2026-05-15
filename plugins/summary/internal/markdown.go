package summaryengine

import (
	"fmt"
	"strings"

	"github.com/edelwud/terraci/pkg/ci"
)

func renderMarkdownSection(section ci.ReportSection) string {
	if section.Kind != ci.ReportSectionKindRendered {
		return ""
	}
	rendered, err := ci.DecodeSection[ci.RenderSection](section)
	if err != nil {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(renderSectionHeader(section))
	for _, block := range rendered.Blocks {
		part := renderMarkdownBlock(block)
		if part == "" {
			continue
		}
		sb.WriteString(part)
		sb.WriteString("\n\n")
	}
	return strings.TrimSpace(sb.String())
}

func renderMarkdownBlock(block ci.RenderBlock) string {
	switch block.Kind {
	case ci.RenderBlockKindText:
		return renderMarkdownTextBlock(block)
	case ci.RenderBlockKindList:
		return renderMarkdownListBlock(block)
	case ci.RenderBlockKindTable:
		return renderMarkdownTableBlock(block)
	case ci.RenderBlockKindDetails:
		return renderMarkdownDetailsBlock(block)
	default:
		return ""
	}
}

func renderMarkdownTextBlock(block ci.RenderBlock) string {
	if block.Text == "" {
		return ""
	}
	var sb strings.Builder
	if block.Title != "" {
		fmt.Fprintf(&sb, "#### %s\n\n", escapeMarkdownText(block.Title))
	}
	sb.WriteString(escapeMarkdownText(block.Text))
	return strings.TrimSpace(sb.String())
}

func renderMarkdownListBlock(block ci.RenderBlock) string {
	if len(block.Items) == 0 {
		return ""
	}
	var sb strings.Builder
	if block.Title != "" {
		fmt.Fprintf(&sb, "#### %s\n\n", escapeMarkdownText(block.Title))
	}
	for _, item := range block.Items {
		fmt.Fprintf(&sb, "- %s\n", escapeMarkdownText(item))
	}
	return strings.TrimSpace(sb.String())
}

func renderMarkdownTableBlock(block ci.RenderBlock) string {
	if block.Table == nil || len(block.Table.Columns) == 0 {
		return ""
	}

	var sb strings.Builder
	if block.Title != "" {
		fmt.Fprintf(&sb, "#### %s\n\n", escapeMarkdownText(block.Title))
	}

	sb.WriteString("|")
	for _, column := range block.Table.Columns {
		fmt.Fprintf(&sb, " %s |", escapeMarkdownTableCell(column))
	}
	sb.WriteString("\n|")
	for range block.Table.Columns {
		sb.WriteString("--------|")
	}
	sb.WriteString("\n")
	for _, row := range block.Table.Rows {
		sb.WriteString("|")
		for idx := range block.Table.Columns {
			cell := ""
			if idx < len(row) {
				cell = row[idx]
			}
			fmt.Fprintf(&sb, " %s |", escapeMarkdownTableCell(cell))
		}
		sb.WriteString("\n")
	}
	return strings.TrimSpace(sb.String())
}

func renderMarkdownDetailsBlock(block ci.RenderBlock) string {
	if block.Details == nil || block.Details.Summary == "" {
		return ""
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "<details>\n<summary>%s</summary>\n\n", escapeHTMLText(block.Details.Summary))
	if block.Details.Body != "" {
		if block.Details.Language != "" {
			fmt.Fprintf(&sb, "```%s\n", block.Details.Language)
			sb.WriteString(block.Details.Body)
			sb.WriteString("\n```\n")
		} else {
			sb.WriteString(block.Details.Body)
			sb.WriteString("\n")
		}
	}
	sb.WriteString("</details>")
	return sb.String()
}

func renderSectionHeader(section ci.ReportSection) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "### %s %s\n\n", reportStatusIcon(section.Status), sectionTitle(section))
	fmt.Fprintf(&sb, "**Status:** %s", section.Status)
	if section.SectionSummary != "" {
		fmt.Fprintf(&sb, " — %s", escapeMarkdownText(section.SectionSummary))
	}
	sb.WriteString("\n\n")
	return sb.String()
}

func reportStatusIcon(status ci.ReportStatus) string {
	switch status {
	case ci.ReportStatusPass:
		return "pass"
	case ci.ReportStatusWarn:
		return "warn"
	case ci.ReportStatusFail:
		return "fail"
	default:
		return string(status)
	}
}
