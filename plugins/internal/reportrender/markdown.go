package reportrender

import (
	"fmt"
	"html"
	"strings"

	"github.com/edelwud/terraci/pkg/ci"
)

// MarkdownReport renders a report using the shared markdown renderer.
func MarkdownReport(report *ci.Report) (string, error) {
	if report == nil {
		return "", nil
	}
	if len(report.Sections) == 0 {
		section, err := ci.NewRenderedSection(ci.RenderedSectionOptions{
			Title:   report.Title,
			Summary: report.Summary,
			Status:  report.Status,
		})
		if err != nil {
			return "", fmt.Errorf("build fallback report section: %w", err)
		}
		return renderMarkdownSectionHeader(section), nil
	}

	var sb strings.Builder
	for i, section := range report.Sections {
		rendered, err := MarkdownSection(section)
		if err != nil {
			return "", fmt.Errorf("render report section %d: %w", i, err)
		}
		if rendered == "" {
			continue
		}
		if sb.Len() > 0 {
			sb.WriteString("\n\n")
		}
		sb.WriteString(rendered)
	}
	return strings.TrimSpace(sb.String()), nil
}

// MarkdownSection renders one render-ready section as markdown.
func MarkdownSection(section ci.ReportSection) (string, error) {
	rendered, err := ci.DecodeRenderSection(section)
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	sb.WriteString(renderMarkdownSectionHeader(section))
	blocks := rendered.Blocks()
	for i := range blocks {
		part := renderMarkdownBlock(blocks[i])
		if part == "" {
			continue
		}
		sb.WriteString(part)
		sb.WriteString("\n\n")
	}
	return strings.TrimSpace(sb.String()), nil
}

func renderMarkdownBlock(block ci.RenderBlock) string {
	switch block.Kind() {
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
	var sb strings.Builder
	if block.Title() != "" {
		fmt.Fprintf(&sb, "#### %s\n\n", escapeMarkdownText(block.Title()))
	}
	sb.WriteString(formatMarkdownValue(block.Text(), false))
	return strings.TrimSpace(sb.String())
}

func renderMarkdownListBlock(block ci.RenderBlock) string {
	var sb strings.Builder
	if block.Title() != "" {
		fmt.Fprintf(&sb, "#### %s\n\n", escapeMarkdownText(block.Title()))
	}
	for _, item := range block.Items() {
		fmt.Fprintf(&sb, "- %s\n", formatMarkdownValue(item, false))
	}
	return strings.TrimSpace(sb.String())
}

func renderMarkdownTableBlock(block ci.RenderBlock) string {
	var sb strings.Builder
	if block.Title() != "" {
		fmt.Fprintf(&sb, "#### %s\n\n", escapeMarkdownText(block.Title()))
	}

	table := block.Table()
	columns := table.Columns()
	rows := table.Rows()
	sb.WriteString("|")
	for _, column := range columns {
		fmt.Fprintf(&sb, " %s |", formatMarkdownValue(column.Title(), true))
	}
	sb.WriteString("\n|")
	for range columns {
		sb.WriteString("--------|")
	}
	sb.WriteString("\n")
	for _, row := range rows {
		cells := row.Cells()
		sb.WriteString("|")
		for idx := range columns {
			cell := ""
			if idx < len(cells) {
				cell = formatMarkdownValue(cells[idx], true)
			}
			fmt.Fprintf(&sb, " %s |", cell)
		}
		sb.WriteString("\n")
	}
	return strings.TrimSpace(sb.String())
}

func renderMarkdownDetailsBlock(block ci.RenderBlock) string {
	details := block.Details()
	var sb strings.Builder
	fmt.Fprintf(&sb, "<details>\n<summary>%s</summary>\n\n", escapeHTMLText(details.Summary()))
	if details.Body() != "" {
		if details.Language() != "" {
			fmt.Fprintf(&sb, "```%s\n", details.Language())
			sb.WriteString(details.Body())
			sb.WriteString("\n```\n")
		} else {
			sb.WriteString(details.Body())
			sb.WriteString("\n")
		}
	}
	sb.WriteString("</details>")
	return sb.String()
}

func renderMarkdownSectionHeader(section ci.ReportSection) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "### %s %s\n\n", StatusLabel(section.Status()), escapeMarkdownText(sectionTitle(section)))
	if section.Summary() != "" {
		sb.WriteString(escapeMarkdownText(section.Summary()))
		sb.WriteString("\n\n")
	}
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

func escapeHTMLText(s string) string {
	return html.EscapeString(escapeMarkdownText(s))
}
