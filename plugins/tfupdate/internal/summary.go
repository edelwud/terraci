package tfupdateengine

import "github.com/edelwud/terraci/plugins/tfupdate/internal/domain"

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
		case domain.StatusError:
			summary.Errors++
		case domain.StatusSkipped:
			summary.Skipped++
		case domain.StatusUpdateAvailable:
			summary.UpdatesAvailable++
		case domain.StatusApplied:
			summary.UpdatesApplied++
		case domain.StatusUpToDate:
		}
	}

	for i := range result.Providers {
		summary.TotalChecked++
		provider := result.Providers[i]
		switch provider.Status {
		case domain.StatusError:
			summary.Errors++
		case domain.StatusSkipped:
			summary.Skipped++
		case domain.StatusUpdateAvailable:
			summary.UpdatesAvailable++
		case domain.StatusApplied:
			summary.UpdatesApplied++
		case domain.StatusUpToDate:
		}
	}

	return summary
}
