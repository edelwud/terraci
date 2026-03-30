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

// ModuleVersionUpdate represents a version check for a Terraform module dependency.
type ModuleVersionUpdate struct {
	ModulePath     string       `json:"module_path"`
	CallName       string       `json:"call_name"`
	Source         string       `json:"source"`
	CurrentVersion string       `json:"current_version"`
	LatestVersion  string       `json:"latest_version"`
	BumpedVersion  string       `json:"bumped_version,omitempty"`
	Constraint     string       `json:"constraint"`
	Status         UpdateStatus `json:"status"`
	Issue          string       `json:"issue,omitempty"`
	File           string       `json:"file,omitempty"`
}

// DisplayCurrent returns the best current-version representation for humans.
func (u ModuleVersionUpdate) DisplayCurrent() string {
	if u.CurrentVersion != "" {
		return u.CurrentVersion
	}
	return u.Constraint
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

// IncludedInUpdateLogs returns true when this item should appear in grouped update output.
func (u ModuleVersionUpdate) IncludedInUpdateLogs() bool {
	return u.IsUpdatable()
}

// ProviderVersionUpdate represents a version check for a Terraform provider.
type ProviderVersionUpdate struct {
	ModulePath     string       `json:"module_path"`
	ProviderName   string       `json:"provider_name"`
	ProviderSource string       `json:"provider_source"`
	CurrentVersion string       `json:"current_version"`
	LatestVersion  string       `json:"latest_version"`
	BumpedVersion  string       `json:"bumped_version,omitempty"`
	Constraint     string       `json:"constraint"`
	Status         UpdateStatus `json:"status"`
	Issue          string       `json:"issue,omitempty"`
	File           string       `json:"file,omitempty"`
}

// DisplayCurrent returns the best current-version representation for humans.
func (u ProviderVersionUpdate) DisplayCurrent() string {
	if u.CurrentVersion != "" {
		return u.CurrentVersion
	}
	return u.Constraint
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

// IncludedInUpdateLogs returns true when this item should appear in grouped update output.
func (u ProviderVersionUpdate) IncludedInUpdateLogs() bool {
	return u.IsUpdatable()
}
