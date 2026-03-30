package updateengine

// UpdateSummary provides aggregated counts.
type UpdateSummary struct {
	TotalChecked     int `json:"total_checked"`
	UpdatesAvailable int `json:"updates_available"`
	UpdatesApplied   int `json:"updates_applied"`
	Skipped          int `json:"skipped"`
	Errors           int `json:"errors"`
}

// BuildUpdateSummary recalculates derived counters from the current result items.
func BuildUpdateSummary(result *UpdateResult) UpdateSummary {
	summary := result.Summary

	for i := range result.Modules {
		summary.TotalChecked++
		module := result.Modules[i]
		switch {
		case module.Error != "":
			summary.Errors++
		case module.Skipped:
			summary.Skipped++
		case module.UpdateAvailable:
			summary.UpdatesAvailable++
		}
		if module.Applied {
			summary.UpdatesApplied++
		}
	}

	for i := range result.Providers {
		summary.TotalChecked++
		provider := result.Providers[i]
		switch {
		case provider.Error != "":
			summary.Errors++
		case provider.Skipped:
			summary.Skipped++
		case provider.UpdateAvailable:
			summary.UpdatesAvailable++
		}
		if provider.Applied {
			summary.UpdatesApplied++
		}
	}

	return summary
}
