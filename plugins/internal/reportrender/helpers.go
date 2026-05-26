package reportrender

import "github.com/edelwud/terraci/pkg/ci"

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
