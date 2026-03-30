package updateengine

// UpdateResult contains all version check results.
type UpdateResult struct {
	Modules   []ModuleVersionUpdate   `json:"modules,omitempty"`
	Providers []ProviderVersionUpdate `json:"providers,omitempty"`
	Summary   UpdateSummary           `json:"summary"`
}

// NewUpdateResult constructs an empty update result accumulator.
func NewUpdateResult() *UpdateResult {
	return &UpdateResult{}
}

// RecordError adds a non-item-specific operational error to the summary.
func (r *UpdateResult) RecordError() {
	r.Summary.Errors++
}
