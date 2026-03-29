package updateengine

// UpdateResult contains all version check results.
type UpdateResult struct {
	Modules   []ModuleVersionUpdate   `json:"modules,omitempty"`
	Providers []ProviderVersionUpdate `json:"providers,omitempty"`
	Summary   UpdateSummary           `json:"summary"`
}

// ModuleVersionUpdate represents a version check for a Terraform module dependency.
type ModuleVersionUpdate struct {
	ModulePath     string `json:"module_path"`
	CallName       string `json:"call_name"`
	Source         string `json:"source"`
	CurrentVersion string `json:"current_version"`
	LatestVersion  string `json:"latest_version"`
	BumpedVersion  string `json:"bumped_version,omitempty"`
	Constraint     string `json:"constraint"`
	Updated        bool   `json:"updated"`
	Skipped        bool   `json:"skipped"`
	SkipReason     string `json:"skip_reason,omitempty"`
	File           string `json:"file,omitempty"`
}

// ProviderVersionUpdate represents a version check for a Terraform provider.
type ProviderVersionUpdate struct {
	ModulePath     string `json:"module_path"`
	ProviderName   string `json:"provider_name"`
	ProviderSource string `json:"provider_source"`
	CurrentVersion string `json:"current_version"`
	LatestVersion  string `json:"latest_version"`
	BumpedVersion  string `json:"bumped_version,omitempty"`
	Constraint     string `json:"constraint"`
	Updated        bool   `json:"updated"`
	Skipped        bool   `json:"skipped"`
	SkipReason     string `json:"skip_reason,omitempty"`
	File           string `json:"file,omitempty"`
}

// UpdateSummary provides aggregated counts.
type UpdateSummary struct {
	TotalChecked     int `json:"total_checked"`
	UpdatesAvailable int `json:"updates_available"`
	UpdatesApplied   int `json:"updates_applied"`
	Skipped          int `json:"skipped"`
	Errors           int `json:"errors"`
}
