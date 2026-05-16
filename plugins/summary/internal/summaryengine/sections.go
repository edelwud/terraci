package summaryengine

import (
	"fmt"
	"sort"

	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/plugins/internal/reportrender"
)

// BuildSummarySections builds the filtered summary view from plan results and render-ready plugin reports.
func BuildSummarySections(plans []ci.PlanResult, reports []*ci.Report) ([]ci.ReportSection, error) {
	return BuildSummarySectionsWithOptions(plans, reports, true)
}

// BuildSummarySectionsWithOptions builds the filtered summary view with explicit rendering options.
func BuildSummarySectionsWithOptions(plans []ci.PlanResult, reports []*ci.Report, includeDetails bool) ([]ci.ReportSection, error) {
	sections := make([]ci.ReportSection, 0, 1+len(plans)+len(reports))
	overview, err := buildSummaryHeaderSection(plans, reports)
	if err != nil {
		return nil, err
	}
	sections = append(sections, overview)
	terraformSections, err := buildTerraformPlanSections(plans, includeDetails)
	if err != nil {
		return nil, err
	}
	sections = append(sections, terraformSections...)
	for _, report := range reports {
		reportSections, err := filteredReportSectionsE(report)
		if err != nil {
			return nil, err
		}
		sections = append(sections, reportSections...)
	}
	return sections, nil
}

func buildSummaryHeaderSection(plans []ci.PlanResult, reports []*ci.Report) (ci.ReportSection, error) {
	stats := calculateStats(plans)
	items := make([]string, 0, len(reports))
	for _, report := range reports {
		if report == nil {
			continue
		}
		if len(report.Sections) == 0 {
			item := fmt.Sprintf("%s %s", reportStatusIcon(report.Status), report.Title)
			if report.Summary != "" {
				item += ": " + report.Summary
			}
			items = append(items, item)
			continue
		}
		for _, section := range report.Sections {
			item := fmt.Sprintf("%s %s", reportStatusIcon(section.Status), sectionTitle(section))
			if section.SectionSummary != "" {
				item += ": " + section.SectionSummary
			}
			items = append(items, item)
		}
	}

	blocks := make([]ci.RenderBlock, 0, 1)
	if len(items) > 0 {
		blocks = append(blocks, ci.RenderListBlock("", items))
	}
	return encodeRenderSection(
		"Summary",
		renderStats(stats),
		overallSummaryStatus(plans, reports),
		blocks...,
	)
}

func overallSummaryStatus(plans []ci.PlanResult, reports []*ci.Report) ci.ReportStatus {
	for i := range plans {
		if plans[i].Status == ci.PlanStatusFailed {
			return ci.ReportStatusFail
		}
	}
	for _, report := range reports {
		if report == nil {
			continue
		}
		if report.Status == ci.ReportStatusFail {
			return ci.ReportStatusFail
		}
	}
	for i := range plans {
		if plans[i].Status == ci.PlanStatusChanges {
			return ci.ReportStatusWarn
		}
	}
	for _, report := range reports {
		if report == nil {
			continue
		}
		if report.Status == ci.ReportStatusWarn {
			return ci.ReportStatusWarn
		}
	}
	return ci.ReportStatusPass
}

func reportStatusIcon(status ci.ReportStatus) string {
	return reportrender.StatusLabel(status)
}

func buildTerraformPlanSections(plans []ci.PlanResult, includeDetails bool) ([]ci.ReportSection, error) {
	byEnv := groupByEnvironment(plans)
	envOrder := sortedKeys(byEnv)
	sections := make([]ci.ReportSection, 0, len(envOrder))
	for _, env := range envOrder {
		envPlans := visibleEnvironmentPlans(byEnv[env])
		if len(envPlans) == 0 {
			continue
		}

		sort.Slice(envPlans, func(i, j int) bool {
			return envPlans[i].ModuleID < envPlans[j].ModuleID
		})

		rows := make([][]string, 0, len(envPlans))
		blocks := make([]ci.RenderBlock, 0, 1+len(envPlans))
		status := ci.ReportStatusWarn
		for i := range envPlans {
			plan := envPlans[i]
			if plan.Status == ci.PlanStatusFailed {
				status = ci.ReportStatusFail
			}
			rows = append(rows, []string{statusIcon(plan.Status), plan.ModuleID, planSummary(plan)})
			if includeDetails {
				if details := planDetailsBody(plan); details != "" {
					blocks = append(blocks, ci.RenderDetailsBlock(planDetailsTitle(plan), details, ""))
				}
			}
		}
		blocks = append([]ci.RenderBlock{ci.RenderTableBlock("", []string{"Status", columnModule, "Summary"}, rows)}, blocks...)

		section, err := encodeRenderSection(
			fmt.Sprintf("Environment: `%s`", env),
			fmt.Sprintf("%d actionable modules", len(rows)),
			status,
			blocks...,
		)
		if err != nil {
			return nil, err
		}
		sections = append(sections, section)
	}
	return sections, nil
}
