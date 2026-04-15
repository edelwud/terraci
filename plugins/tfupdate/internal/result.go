package tfupdateengine

import "github.com/edelwud/terraci/plugins/tfupdate/internal/domain"

// UpdateResult contains all version check results.
type UpdateResult struct {
	Modules    []domain.ModuleVersionUpdate   `json:"modules,omitempty"`
	Providers  []domain.ProviderVersionUpdate `json:"providers,omitempty"`
	LockSync   []domain.LockSyncPlan          `json:"-"`
	Summary    UpdateSummary                  `json:"summary"`
	baseErrors int
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
	r.baseErrors++
}

// AddModuleUpdate appends a checked module outcome.
func (b *UpdateResultBuilder) AddModuleUpdate(update domain.ModuleVersionUpdate) {
	b.result.Modules = append(b.result.Modules, update)
}

// AddProviderUpdate appends a checked provider outcome.
func (b *UpdateResultBuilder) AddProviderUpdate(update domain.ProviderVersionUpdate) {
	b.result.Providers = append(b.result.Providers, update)
}

// AddLockSyncPlan appends an explicit lock synchronization plan.
func (b *UpdateResultBuilder) AddLockSyncPlan(plan domain.LockSyncPlan) {
	if len(plan.Providers) == 0 {
		return
	}
	b.result.LockSync = append(b.result.LockSync, plan)
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
