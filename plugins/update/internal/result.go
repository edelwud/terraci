package updateengine

// UpdateResult contains all version check results.
type UpdateResult struct {
	Modules   []ModuleVersionUpdate   `json:"modules,omitempty"`
	Providers []ProviderVersionUpdate `json:"providers,omitempty"`
	Summary   UpdateSummary           `json:"summary"`
}

// UpdateResultBuilder accumulates update outcomes and summary-level errors.
type UpdateResultBuilder struct {
	result *UpdateResult
}

// NewUpdateResult constructs an empty update result accumulator.
func NewUpdateResult() *UpdateResult {
	return &UpdateResult{}
}

// NewUpdateResultBuilder constructs an empty result builder.
func NewUpdateResultBuilder() *UpdateResultBuilder {
	return &UpdateResultBuilder{result: NewUpdateResult()}
}

// RecordError adds a non-item-specific operational error to the summary.
func (r *UpdateResult) RecordError() {
	r.Summary.Errors++
}

// AddModuleUpdate appends a checked module outcome.
func (b *UpdateResultBuilder) AddModuleUpdate(update ModuleVersionUpdate) {
	b.result.Modules = append(b.result.Modules, update)
}

// AddProviderUpdate appends a checked provider outcome.
func (b *UpdateResultBuilder) AddProviderUpdate(update ProviderVersionUpdate) {
	b.result.Providers = append(b.result.Providers, update)
}

// RecordError adds a non-item-specific operational error to the underlying result.
func (b *UpdateResultBuilder) RecordError() {
	b.result.RecordError()
}

// Result returns the underlying mutable result.
func (b *UpdateResultBuilder) Result() *UpdateResult {
	return b.result
}

// Build finalizes derived fields and returns the accumulated result.
func (b *UpdateResultBuilder) Build() *UpdateResult {
	b.result.Summary = BuildUpdateSummary(b.result)
	return b.result
}
