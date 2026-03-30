package updateengine

// UpdateStatus represents the lifecycle state of a dependency update check.
type UpdateStatus string

const (
	StatusUpToDate        UpdateStatus = "up_to_date"
	StatusUpdateAvailable UpdateStatus = "update_available"
	StatusApplied         UpdateStatus = "applied"
	StatusSkipped         UpdateStatus = "skipped"
	StatusError           UpdateStatus = "error"
)

const (
	statusLabelUpToDate        = "up to date"
	statusLabelUpdateAvailable = "update available"
	statusLabelApplied         = "applied"
)

// ModuleDependency describes a discovered Terraform module dependency before any checks are run.
type ModuleDependency struct {
	ModulePath string `json:"module_path"`
	CallName   string `json:"call_name"`
	Source     string `json:"source"`
	Constraint string `json:"constraint"`
}

// ProviderDependency describes a discovered Terraform provider dependency before any checks are run.
type ProviderDependency struct {
	ModulePath     string `json:"module_path"`
	ProviderName   string `json:"provider_name"`
	ProviderSource string `json:"provider_source"`
	Constraint     string `json:"constraint"`
}

// ModuleVersionUpdate represents the outcome of checking a Terraform module dependency.
type ModuleVersionUpdate struct {
	Dependency     ModuleDependency `json:"dependency"`
	CurrentVersion string           `json:"current_version"`
	LatestVersion  string           `json:"latest_version"`
	BumpedVersion  string           `json:"bumped_version,omitempty"`
	Status         UpdateStatus     `json:"status"`
	Issue          string           `json:"issue,omitempty"`
	File           string           `json:"file,omitempty"`
}

// NewModuleVersionUpdate creates an up-to-date module outcome from a discovered dependency.
func NewModuleVersionUpdate(dep ModuleDependency) ModuleVersionUpdate {
	return ModuleVersionUpdate{
		Dependency: dep,
		Status:     StatusUpToDate,
	}
}

// ModulePath returns the Terraform module path containing the dependency.
func (u ModuleVersionUpdate) ModulePath() string {
	return u.Dependency.ModulePath
}

// CallName returns the Terraform module call name.
func (u ModuleVersionUpdate) CallName() string {
	return u.Dependency.CallName
}

// Source returns the dependency source reference.
func (u ModuleVersionUpdate) Source() string {
	return u.Dependency.Source
}

// Constraint returns the declared version constraint.
func (u ModuleVersionUpdate) Constraint() string {
	return u.Dependency.Constraint
}

// DisplayCurrent returns the best current-version representation for humans.
func (u ModuleVersionUpdate) DisplayCurrent() string {
	if u.CurrentVersion != "" {
		return u.CurrentVersion
	}
	return u.Constraint()
}

// StatusLabel returns a human-readable state for reporting surfaces.
func (u ModuleVersionUpdate) StatusLabel() string {
	switch u.Status {
	case StatusSkipped:
		return "skipped: " + u.Issue
	case StatusError:
		return "error: " + u.Issue
	case StatusApplied:
		return statusLabelApplied
	case StatusUpdateAvailable:
		return statusLabelUpdateAvailable
	case StatusUpToDate:
		return statusLabelUpToDate
	}
	return statusLabelUpToDate
}

// IsUpdatable returns true when the dependency can be applied or surfaced in update logs.
func (u ModuleVersionUpdate) IsUpdatable() bool {
	return u.Status == StatusUpdateAvailable || u.Status == StatusApplied
}

// IsApplyPending returns true when the module update is ready to be written.
func (u ModuleVersionUpdate) IsApplyPending() bool {
	return u.Status == StatusUpdateAvailable
}

// MarkApplied transitions the module outcome into an applied state.
func (u ModuleVersionUpdate) MarkApplied() ModuleVersionUpdate {
	u.Status = StatusApplied
	u.Issue = ""
	return u
}

// MarkError transitions the module outcome into an error state with context.
func (u ModuleVersionUpdate) MarkError(issue string) ModuleVersionUpdate {
	u.Status = StatusError
	u.Issue = issue
	return u
}

// IncludedInUpdateLogs returns true when this item should appear in grouped update output.
func (u ModuleVersionUpdate) IncludedInUpdateLogs() bool {
	return u.IsUpdatable()
}

// ProviderVersionUpdate represents the outcome of checking a Terraform provider dependency.
type ProviderVersionUpdate struct {
	Dependency     ProviderDependency `json:"dependency"`
	CurrentVersion string             `json:"current_version"`
	LatestVersion  string             `json:"latest_version"`
	BumpedVersion  string             `json:"bumped_version,omitempty"`
	Status         UpdateStatus       `json:"status"`
	Issue          string             `json:"issue,omitempty"`
	File           string             `json:"file,omitempty"`
}

// NewProviderVersionUpdate creates an up-to-date provider outcome from a discovered dependency.
func NewProviderVersionUpdate(dep ProviderDependency) ProviderVersionUpdate {
	return ProviderVersionUpdate{
		Dependency: dep,
		Status:     StatusUpToDate,
	}
}

// ModulePath returns the Terraform module path containing the dependency.
func (u ProviderVersionUpdate) ModulePath() string {
	return u.Dependency.ModulePath
}

// ProviderName returns the Terraform provider local name.
func (u ProviderVersionUpdate) ProviderName() string {
	return u.Dependency.ProviderName
}

// ProviderSource returns the provider source reference.
func (u ProviderVersionUpdate) ProviderSource() string {
	return u.Dependency.ProviderSource
}

// Constraint returns the declared version constraint.
func (u ProviderVersionUpdate) Constraint() string {
	return u.Dependency.Constraint
}

// DisplayCurrent returns the best current-version representation for humans.
func (u ProviderVersionUpdate) DisplayCurrent() string {
	if u.CurrentVersion != "" {
		return u.CurrentVersion
	}
	return u.Constraint()
}

// StatusLabel returns a human-readable state for reporting surfaces.
func (u ProviderVersionUpdate) StatusLabel() string {
	switch u.Status {
	case StatusSkipped:
		return "skipped: " + u.Issue
	case StatusError:
		return "error: " + u.Issue
	case StatusApplied:
		return statusLabelApplied
	case StatusUpdateAvailable:
		return statusLabelUpdateAvailable
	case StatusUpToDate:
		return statusLabelUpToDate
	}
	return statusLabelUpToDate
}

// IsUpdatable returns true when the dependency can be applied or surfaced in update logs.
func (u ProviderVersionUpdate) IsUpdatable() bool {
	return u.Status == StatusUpdateAvailable || u.Status == StatusApplied
}

// IsApplyPending returns true when the provider update is ready to be written.
func (u ProviderVersionUpdate) IsApplyPending() bool {
	return u.Status == StatusUpdateAvailable
}

// MarkApplied transitions the provider outcome into an applied state.
func (u ProviderVersionUpdate) MarkApplied() ProviderVersionUpdate {
	u.Status = StatusApplied
	u.Issue = ""
	return u
}

// MarkError transitions the provider outcome into an error state with context.
func (u ProviderVersionUpdate) MarkError(issue string) ProviderVersionUpdate {
	u.Status = StatusError
	u.Issue = issue
	return u
}

// IncludedInUpdateLogs returns true when this item should appear in grouped update output.
func (u ProviderVersionUpdate) IncludedInUpdateLogs() bool {
	return u.IsUpdatable()
}
