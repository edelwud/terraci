package tfupdateengine

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
	summary := UpdateSummary{
		Errors: result.baseErrors,
	}

	for i := range result.Modules {
		summary.TotalChecked++
		module := result.Modules[i]
		switch module.Status {
		case StatusError:
			summary.Errors++
		case StatusSkipped:
			summary.Skipped++
		case StatusUpdateAvailable:
			summary.UpdatesAvailable++
		case StatusApplied:
			summary.UpdatesApplied++
		case StatusUpToDate:
		}
	}

	for i := range result.Providers {
		summary.TotalChecked++
		provider := result.Providers[i]
		switch provider.Status {
		case StatusError:
			summary.Errors++
		case StatusSkipped:
			summary.Skipped++
		case StatusUpdateAvailable:
			summary.UpdatesAvailable++
		case StatusApplied:
			summary.UpdatesApplied++
		case StatusUpToDate:
		}
	}

	return summary
}
