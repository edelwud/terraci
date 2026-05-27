package reportrender

import (
	"fmt"
	"math"
	"strings"

	"github.com/edelwud/terraci/pkg/ci"
)

// StatusLabel returns the stable human-readable status label used by report renderers.
func StatusLabel(status ci.ReportStatus) string {
	switch status {
	case ci.ReportStatusPass:
		return "Passed"
	case ci.ReportStatusWarn:
		return "Warning"
	case ci.ReportStatusFail:
		return "Failed"
	default:
		return string(status)
	}
}

func sectionTitle(section ci.ReportSection) string {
	if section.Title() != "" {
		return section.Title()
	}
	if section.Kind() == ci.ReportSectionKindRendered {
		return "Report"
	}
	return string(section.Kind())
}

func formatPlainValue(value ci.RenderValue) string {
	switch value.Kind() {
	case ci.RenderValueKindText, ci.RenderValueKindCode, ci.RenderValueKindLabel, ci.RenderValueKindModulePath, ci.RenderValueKindResourceAddress:
		return value.Text()
	case ci.RenderValueKindStatus:
		return StatusLabel(value.Status())
	case ci.RenderValueKindMoney:
		return formatMoney(value.Amount(), value.Unit(), false)
	case ci.RenderValueKindMoneyDelta:
		return formatMoney(value.Amount(), value.Unit(), true)
	case ci.RenderValueKindInline:
		var sb strings.Builder
		for _, part := range value.Parts() {
			sb.WriteString(formatPlainValue(part))
		}
		return sb.String()
	default:
		return ""
	}
}

func formatMarkdownValue(value ci.RenderValue, tableCell bool) string {
	switch value.Kind() {
	case ci.RenderValueKindCode:
		return "`" + escapeBackticks(escapeMarkdownValueText(value.Text(), tableCell)) + "`"
	case ci.RenderValueKindInline:
		var sb strings.Builder
		for _, part := range value.Parts() {
			sb.WriteString(formatMarkdownValue(part, tableCell))
		}
		return sb.String()
	case ci.RenderValueKindText,
		ci.RenderValueKindStatus,
		ci.RenderValueKindLabel,
		ci.RenderValueKindMoney,
		ci.RenderValueKindMoneyDelta,
		ci.RenderValueKindModulePath,
		ci.RenderValueKindResourceAddress:
		return escapeMarkdownValueText(formatPlainValue(value), tableCell)
	default:
		return ""
	}
}

func escapeMarkdownValueText(s string, tableCell bool) string {
	if tableCell {
		return escapeMarkdownTableCell(s)
	}
	return escapeMarkdownText(s)
}

func escapeBackticks(s string) string {
	return strings.ReplaceAll(s, "`", "\\`")
}

func formatMoney(amount float64, unit ci.RenderMoneyUnit, signed bool) string {
	prefix := ""
	value := amount
	if signed {
		switch {
		case isZeroCost(amount):
			value = 0
		case amount > 0:
			prefix = "+"
		default:
			prefix = "-"
			value = -amount
		}
	}

	rendered := prefix + formatMoneyAmount(value)
	if unit != "" {
		rendered += "/" + string(unit)
	}
	return rendered
}

func formatMoneyAmount(amount float64) string {
	if isZeroCost(amount) {
		return "$0"
	}
	if amount < 0 {
		return "-" + formatMoneyAmount(-amount)
	}
	if amount < 0.01 {
		return "<$0.01"
	}
	if amount >= 1000 {
		return fmt.Sprintf("$%.0f", amount)
	}
	if amount >= 1 {
		return fmt.Sprintf("$%.2f", amount)
	}
	return fmt.Sprintf("$%.4f", amount)
}

func isZeroCost(amount float64) bool {
	return math.Abs(amount) < 0.0000001
}
