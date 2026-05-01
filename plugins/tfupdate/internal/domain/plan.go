package domain

import (
	"github.com/edelwud/terraci/plugins/tfupdate/internal/registrymeta"
	"github.com/edelwud/terraci/plugins/tfupdate/internal/versionkit"
)

type VersionPolicy struct {
	Bump string
	Pin  bool
}

type RegistrySelection struct {
	Hostname string
	Source   string
}

type VersionConstraintSet struct {
	Raw         []string
	Constraints []versionkit.Constraint
}

type TransitiveProviderConstraint struct {
	ModuleSource string
	ProviderDep  registrymeta.ModuleProviderDep
}

type DependencyCandidate struct {
	Version      string
	ProviderDeps []registrymeta.ModuleProviderDep
}

type ModuleResolution struct {
	Dependency   ModuleDependency
	Registry     RegistrySelection
	File         string
	Status       UpdateStatus
	Issue        string
	Current      string
	Selected     string
	Latest       string
	ProviderDeps []registrymeta.ModuleProviderDep
	Candidates   []DependencyCandidate
}

type ProviderResolution struct {
	Dependency       ProviderDependency
	Registry         RegistrySelection
	File             string
	Status           UpdateStatus
	Issue            string
	Current          string
	Selected         string
	Latest           string
	Locked           bool
	LockedSource     string
	LockedConstraint string
	Constraints      VersionConstraintSet
}

type LockProviderSync struct {
	ProviderSource string
	Version        string
	Constraint     string
	TerraformFile  string
}

type PlanResult struct {
	ModulePath string
	Modules    []ModuleResolution
	Providers  []ProviderResolution
	LockSync   LockSyncPlan
}

type LockSyncPlan struct {
	ModulePath string
	Providers  []LockProviderSync
}
