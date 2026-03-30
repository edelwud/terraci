package updateengine

// ModuleVersionUpdate represents a version check for a Terraform module dependency.
type ModuleVersionUpdate struct {
	ModulePath      string `json:"module_path"`
	CallName        string `json:"call_name"`
	Source          string `json:"source"`
	CurrentVersion  string `json:"current_version"`
	LatestVersion   string `json:"latest_version"`
	BumpedVersion   string `json:"bumped_version,omitempty"`
	Constraint      string `json:"constraint"`
	UpdateAvailable bool   `json:"update_available"`
	Applied         bool   `json:"applied"`
	Skipped         bool   `json:"skipped"`
	SkipReason      string `json:"skip_reason,omitempty"`
	Error           string `json:"error,omitempty"`
	File            string `json:"file,omitempty"`
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
	switch {
	case u.Skipped:
		return "skipped: " + u.SkipReason
	case u.Error != "":
		return "error: " + u.Error
	case u.Applied:
		return "applied"
	case u.UpdateAvailable:
		return "update available"
	default:
		return "up to date"
	}
}

// IncludedInUpdateLogs returns true when this item should appear in grouped update output.
func (u ModuleVersionUpdate) IncludedInUpdateLogs() bool {
	return !u.Skipped && u.UpdateAvailable
}

// ProviderVersionUpdate represents a version check for a Terraform provider.
type ProviderVersionUpdate struct {
	ModulePath      string `json:"module_path"`
	ProviderName    string `json:"provider_name"`
	ProviderSource  string `json:"provider_source"`
	CurrentVersion  string `json:"current_version"`
	LatestVersion   string `json:"latest_version"`
	BumpedVersion   string `json:"bumped_version,omitempty"`
	Constraint      string `json:"constraint"`
	UpdateAvailable bool   `json:"update_available"`
	Applied         bool   `json:"applied"`
	Skipped         bool   `json:"skipped"`
	SkipReason      string `json:"skip_reason,omitempty"`
	Error           string `json:"error,omitempty"`
	File            string `json:"file,omitempty"`
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
	switch {
	case u.Skipped:
		return "skipped: " + u.SkipReason
	case u.Error != "":
		return "error: " + u.Error
	case u.Applied:
		return "applied"
	case u.UpdateAvailable:
		return "update available"
	default:
		return "up to date"
	}
}

// IncludedInUpdateLogs returns true when this item should appear in grouped update output.
func (u ProviderVersionUpdate) IncludedInUpdateLogs() bool {
	return !u.Skipped && u.UpdateAvailable
}
