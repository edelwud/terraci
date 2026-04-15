package domain

import "github.com/edelwud/terraci/plugins/tfupdate/internal/registrymeta"

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
	Dependency     ModuleDependency                 `json:"dependency"`
	CurrentVersion string                           `json:"current_version"`
	LatestVersion  string                           `json:"latest_version"`
	BumpedVersion  string                           `json:"bumped_version,omitempty"`
	Status         UpdateStatus                     `json:"status"`
	Issue          string                           `json:"issue,omitempty"`
	File           string                           `json:"file,omitempty"`
	ProviderDeps   []registrymeta.ModuleProviderDep `json:"provider_deps,omitempty"`
}

// NewModuleVersionUpdate creates an up-to-date module outcome from a discovered dependency.
func NewModuleVersionUpdate(dep ModuleDependency) ModuleVersionUpdate {
	return ModuleVersionUpdate{
		Dependency: dep,
		Status:     StatusUpToDate,
	}
}

func (u ModuleVersionUpdate) ModulePath() string {
	return u.Dependency.ModulePath
}

func (u ModuleVersionUpdate) CallName() string {
	return u.Dependency.CallName
}

func (u ModuleVersionUpdate) Source() string {
	return u.Dependency.Source
}

func (u ModuleVersionUpdate) Constraint() string {
	return u.Dependency.Constraint
}

func (u ModuleVersionUpdate) DisplayCurrent() string {
	return displayCurrent(u.Constraint(), u.CurrentVersion)
}

func (u ModuleVersionUpdate) DisplayAvailable() string {
	return u.BumpedVersion
}

func (u ModuleVersionUpdate) DisplayLatest() string {
	return displayLatest(u.LatestVersion, u.BumpedVersion)
}

func (u ModuleVersionUpdate) IsApplied() bool {
	return u.Status == StatusApplied
}

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

func (u ModuleVersionUpdate) IsUpdatable() bool {
	return u.Status == StatusUpdateAvailable || u.Status == StatusApplied
}

func (u ModuleVersionUpdate) IsApplyPending() bool {
	return u.Status == StatusUpdateAvailable
}

func (u ModuleVersionUpdate) MarkApplied() ModuleVersionUpdate {
	u.Status = StatusApplied
	u.Issue = ""
	return u
}

func (u ModuleVersionUpdate) MarkError(issue string) ModuleVersionUpdate {
	u.Status = StatusError
	u.Issue = issue
	return u
}

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

func (u ProviderVersionUpdate) ModulePath() string {
	return u.Dependency.ModulePath
}

func (u ProviderVersionUpdate) ProviderName() string {
	return u.Dependency.ProviderName
}

func (u ProviderVersionUpdate) ProviderSource() string {
	return u.Dependency.ProviderSource
}

func (u ProviderVersionUpdate) Constraint() string {
	return u.Dependency.Constraint
}

func (u ProviderVersionUpdate) DisplayCurrent() string {
	return displayCurrent(u.Constraint(), u.CurrentVersion)
}

func (u ProviderVersionUpdate) DisplayAvailable() string {
	return u.BumpedVersion
}

func (u ProviderVersionUpdate) DisplayLatest() string {
	return displayLatest(u.LatestVersion, u.BumpedVersion)
}

func (u ProviderVersionUpdate) IsApplied() bool {
	return u.Status == StatusApplied
}

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

func (u ProviderVersionUpdate) IsUpdatable() bool {
	return u.Status == StatusUpdateAvailable || u.Status == StatusApplied
}

func (u ProviderVersionUpdate) IsApplyPending() bool {
	return u.Status == StatusUpdateAvailable
}

func (u ProviderVersionUpdate) MarkApplied() ProviderVersionUpdate {
	u.Status = StatusApplied
	u.Issue = ""
	return u
}

func (u ProviderVersionUpdate) MarkError(issue string) ProviderVersionUpdate {
	u.Status = StatusError
	u.Issue = issue
	return u
}

func (u ProviderVersionUpdate) IncludedInUpdateLogs() bool {
	return u.IsUpdatable()
}

func displayCurrent(constraint, resolved string) string {
	if resolved == "" || constraint == resolved {
		return constraint
	}
	if constraint == "" {
		return resolved
	}
	return constraint + " (" + resolved + ")"
}

func displayLatest(latest, bumped string) string {
	if latest == "" || latest == bumped {
		return ""
	}
	return latest
}
