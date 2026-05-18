package summaryengine

import (
	"fmt"

	"github.com/edelwud/terraci/pkg/ci"
)

func filteredReportSectionsE(report *ci.Report) ([]ci.ReportSection, error) {
	if report == nil {
		return nil, nil
	}
	if len(report.Sections) == 0 {
		section, err := encodeRenderSection(report.Title, report.Summary, report.Status)
		if err != nil {
			return nil, fmt.Errorf("build fallback report section: %w", err)
		}
		return []ci.ReportSection{section}, nil
	}

	sections := make([]ci.ReportSection, 0, len(report.Sections))
	for _, section := range report.Sections {
		if section.Kind() != ci.ReportSectionKindRendered {
			return nil, fmt.Errorf(
				"%s report section %q is not render-ready; plugins must publish %q sections",
				report.Producer,
				section.Kind(),
				ci.ReportSectionKindRendered,
			)
		}
		if _, err := ci.DecodeRenderSection(section); err != nil {
			return nil, fmt.Errorf("decode %s report section %s: %w", report.Producer, section.Kind(), err)
		}
		sections = append(sections, section)
	}
	return sections, nil
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
