package summaryengine

import "github.com/edelwud/terraci/pkg/ci"

// EnrichPlansFromReports applies typed report enrichment data to plan rows.
func EnrichPlansFromReports(plans []ci.ModulePlan, reports []*ci.Report) {
	if len(plans) == 0 || len(reports) == 0 {
		return
	}

	byPath := make(map[string]ci.CostChangeRow)
	for _, report := range reports {
		if report == nil {
			continue
		}
		for _, section := range report.Sections {
			if section.Kind != ci.ReportSectionKindCostChanges || section.CostChanges == nil {
				continue
			}
			for _, row := range section.CostChanges.Rows {
				byPath[row.ModulePath] = row
			}
		}
	}

	for i := range plans {
		row, ok := byPath[plans[i].ModulePath]
		if !ok || !row.HasCost {
			continue
		}
		plans[i].CostBefore = row.Before
		plans[i].CostAfter = row.After
		plans[i].CostDiff = row.Diff
		plans[i].HasCost = row.HasCost
	}
}
