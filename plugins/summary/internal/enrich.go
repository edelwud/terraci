package summaryengine

import "github.com/edelwud/terraci/pkg/ci"

// EnrichPlansFromReports applies typed report enrichment data to plan rows.
func EnrichPlansFromReports(plans []ci.ModulePlan, reports []*ci.Report) {
	if len(plans) == 0 || len(reports) == 0 {
		return
	}

	byPath := make(map[string]ci.EstimateChangeRow)
	for _, report := range reports {
		if report == nil {
			continue
		}
		for _, section := range report.Sections {
			if section.Kind != ci.ReportSectionKindEstimateChanges || section.EstimateChanges == nil {
				continue
			}
			for _, row := range section.EstimateChanges.Rows {
				byPath[row.ModulePath] = row
			}
		}
	}

	for i := range plans {
		row, ok := byPath[plans[i].ModulePath]
		if !ok || !row.HasEstimate {
			continue
		}
		plans[i].EstimateBefore = row.Before
		plans[i].EstimateAfter = row.After
		plans[i].EstimateDiff = row.Diff
		plans[i].HasEstimate = row.HasEstimate
	}
}
