package reportrender

import "github.com/edelwud/terraci/pkg/ci"

// StatusLabel returns the stable human-readable status label used by report renderers.
func StatusLabel(status ci.ReportStatus) string {
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

func sectionTitle(section ci.ReportSection) string {
	if section.Title != "" {
		return section.Title
	}
	if section.Kind == ci.ReportSectionKindRendered {
		return "Report"
	}
	return string(section.Kind)
}
