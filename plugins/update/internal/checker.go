package updateengine

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/caarlos0/log"

	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/parser"
)

const skipReasonIgnored = "ignored by config"

// Checker performs version checks across Terraform modules.
type Checker struct {
	config   *UpdateConfig
	parser   *parser.Parser
	registry RegistryClient
	write    bool
}

// NewChecker creates a new dependency version checker.
func NewChecker(cfg *UpdateConfig, p *parser.Parser, reg RegistryClient, write bool) *Checker {
	return &Checker{
		config:   cfg,
		parser:   p,
		registry: reg,
		write:    write,
	}
}

// Check performs version checks on all provided modules.
func (c *Checker) Check(ctx context.Context, modules []*discovery.Module) (*UpdateResult, error) {
	result := &UpdateResult{}

	for _, mod := range modules {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		parsed, err := c.parser.ParseModule(ctx, mod.Path)
		if err != nil {
			log.WithField("module", mod.RelativePath).WithError(err).Warn("failed to parse module")
			result.Summary.Errors++
			continue
		}

		if c.config.ShouldCheckProviders() {
			c.checkProviders(ctx, mod, parsed, result)
		}

		if c.config.ShouldCheckModules() {
			c.checkModules(ctx, mod, parsed, result)
		}
	}

	if c.write {
		c.applyUpdates(result)
	}

	result.Summary = computeSummary(result)
	return result, nil
}

func (c *Checker) checkProviders(ctx context.Context, mod *discovery.Module, parsed *parser.ParsedModule, result *UpdateResult) {
	// Build lock file index: short source ("hashicorp/aws") → LockedProvider.
	lockIndex := buildLockIndex(parsed.LockedProviders)

	for _, rp := range parsed.RequiredProviders {
		update := ProviderVersionUpdate{
			ModulePath:     mod.RelativePath,
			ProviderName:   rp.Name,
			ProviderSource: rp.Source,
			Constraint:     rp.VersionConstraint,
		}

		if rp.Source == "" {
			update.Skipped = true
			update.SkipReason = "no source specified"
			result.Providers = append(result.Providers, update)
			continue
		}

		if c.config.IsIgnored(rp.Source) {
			update.Skipped = true
			update.SkipReason = skipReasonIgnored
			result.Providers = append(result.Providers, update)
			continue
		}

		// Merge lock file data: use locked version as current, fill missing constraint.
		if lp, ok := lockIndex[rp.Source]; ok {
			if lp.Version != "" {
				update.CurrentVersion = lp.Version
			}
			if update.Constraint == "" && lp.Constraints != "" {
				update.Constraint = lp.Constraints
			}
		}

		namespace, typeName, err := ParseProviderSource(rp.Source)
		if err != nil {
			update.Skipped = true
			update.SkipReason = fmt.Sprintf("invalid source: %v", err)
			result.Providers = append(result.Providers, update)
			continue
		}

		versionStrings, err := c.registry.ProviderVersions(ctx, namespace, typeName)
		if err != nil {
			update.Error = fmt.Sprintf("registry error: %v", err)
			result.Providers = append(result.Providers, update)
			continue
		}

		versions := parseVersionList(versionStrings)

		// Determine current version: prefer lock file, fall back to constraint resolution.
		var current Version
		if update.CurrentVersion != "" {
			if v, err := ParseVersion(update.CurrentVersion); err == nil {
				current = v
			}
		}
		if current.IsZero() && update.Constraint != "" {
			constraints, parseErr := ParseConstraints(update.Constraint)
			if parseErr == nil {
				if resolved, found := LatestAllowed(versions, constraints); found {
					current = resolved
					update.CurrentVersion = current.String()
				}
			}
		}

		latest := latestStable(versions)
		if !latest.IsZero() {
			update.LatestVersion = latest.String()
		}

		if current.IsZero() {
			// No way to determine current version — still report latest.
			update.Skipped = true
			update.SkipReason = "cannot determine current version"
			result.Providers = append(result.Providers, update)
			continue
		}

		bumped, ok := LatestByBump(current, versions, c.config.Bump)
		if ok {
			update.File = FindTFFileForProvider(mod.Path, rp.Name)
			update.BumpedVersion = bumped.String()
			update.UpdateAvailable = true
		}

		result.Providers = append(result.Providers, update)
	}
}

// buildLockIndex creates a map from short provider source ("hashicorp/aws")
// to LockedProvider, stripping the "registry.terraform.io/" prefix.
func buildLockIndex(locked []*parser.LockedProvider) map[string]*parser.LockedProvider {
	idx := make(map[string]*parser.LockedProvider, len(locked))
	for _, lp := range locked {
		short := stripRegistryPrefix(lp.Source)
		idx[short] = lp
	}
	return idx
}

// stripRegistryPrefix removes the registry hostname prefix from a lock file source.
// Handles registry.terraform.io/ and registry.opentofu.org/.
func stripRegistryPrefix(source string) string {
	for _, prefix := range []string{"registry.terraform.io/", "registry.opentofu.org/"} {
		if strings.HasPrefix(source, prefix) {
			return source[len(prefix):]
		}
	}
	return source
}

func (c *Checker) checkModules(ctx context.Context, mod *discovery.Module, parsed *parser.ParsedModule, result *UpdateResult) {
	for _, mc := range parsed.ModuleCalls {
		if mc.IsLocal || mc.Version == "" {
			continue
		}

		if !IsRegistrySource(mc.Source) {
			continue
		}

		update := ModuleVersionUpdate{
			ModulePath: mod.RelativePath,
			CallName:   mc.Name,
			Source:     mc.Source,
			Constraint: mc.Version,
		}

		if c.config.IsIgnored(mc.Source) {
			update.Skipped = true
			update.SkipReason = skipReasonIgnored
			result.Modules = append(result.Modules, update)
			continue
		}

		namespace, name, provider, err := ParseModuleSource(mc.Source)
		if err != nil {
			update.Skipped = true
			update.SkipReason = fmt.Sprintf("invalid source: %v", err)
			result.Modules = append(result.Modules, update)
			continue
		}

		versionStrings, err := c.registry.ModuleVersions(ctx, namespace, name, provider)
		if err != nil {
			update.Error = fmt.Sprintf("registry error: %v", err)
			result.Modules = append(result.Modules, update)
			continue
		}

		versions := parseVersionList(versionStrings)

		// Use constraint base version as current (e.g. "~> 5.0" → 5.0.0).
		// This reflects the minimum version the module was pinned to.
		current := versionFromConstraint(mc.Version)
		if !current.IsZero() {
			update.CurrentVersion = current.String()
		}

		latest := latestStable(versions)
		if !latest.IsZero() {
			update.LatestVersion = latest.String()
		}

		bumped, ok := LatestByBump(current, versions, c.config.Bump)
		if ok {
			update.File = FindTFFileForModule(mod.Path, mc.Name)
			update.BumpedVersion = bumped.String()
			update.UpdateAvailable = true
		}

		result.Modules = append(result.Modules, update)
	}
}

func (c *Checker) applyUpdates(result *UpdateResult) {
	for i := range result.Modules {
		u := &result.Modules[i]
		if !u.UpdateAvailable {
			continue
		}
		if u.File == "" {
			u.Error = "failed to locate Terraform file for module update"
			log.WithField("module", u.ModulePath).Warn("failed to locate Terraform file for module update")
			continue
		}
		newConstraint := BumpConstraint(u.Constraint, mustParseVersion(u.BumpedVersion))
		if err := WriteModuleVersion(u.File, u.CallName, newConstraint); err != nil {
			log.WithField("module", u.ModulePath).WithError(err).Warn("failed to write module version")
			u.Error = fmt.Sprintf("write module version: %v", err)
			continue
		}
		u.Applied = true
	}

	for i := range result.Providers {
		u := &result.Providers[i]
		if !u.UpdateAvailable {
			continue
		}
		if u.File == "" {
			u.Error = "failed to locate Terraform file for provider update"
			log.WithField("provider", u.ProviderSource).Warn("failed to locate Terraform file for provider update")
			continue
		}
		newConstraint := BumpConstraint(u.Constraint, mustParseVersion(u.BumpedVersion))
		if err := WriteProviderVersion(u.File, u.ProviderName, newConstraint); err != nil {
			log.WithField("provider", u.ProviderSource).WithError(err).Warn("failed to write provider version")
			u.Error = fmt.Sprintf("write provider version: %v", err)
			continue
		}
		u.Applied = true
	}
}

func parseVersionList(strs []string) []Version {
	versions := make([]Version, 0, len(strs))
	for _, s := range strs {
		v, err := ParseVersion(s)
		if err != nil {
			continue
		}
		versions = append(versions, v)
	}
	return versions
}

func latestStable(versions []Version) Version {
	var best Version
	for _, v := range versions {
		if v.Prerelease != "" {
			continue
		}
		if v.Compare(best) > 0 {
			best = v
		}
	}
	return best
}

// versionFromConstraint extracts the base version from a constraint string.
// E.g. "~> 5.0" → 5.0.0, ">= 1.2.3" → 1.2.3, "5.0" → 5.0.0.
func versionFromConstraint(s string) Version {
	constraints, err := ParseConstraints(s)
	if err != nil || len(constraints) == 0 {
		// Try parsing as a plain version.
		v, _ := ParseVersion(s) //nolint:errcheck // best-effort
		return v
	}
	return constraints[0].Version
}

func mustParseVersion(s string) Version {
	v, err := ParseVersion(s)
	if err != nil {
		return Version{}
	}
	return v
}

func computeSummary(result *UpdateResult) UpdateSummary {
	s := result.Summary
	for i := range result.Modules {
		s.TotalChecked++
		switch {
		case result.Modules[i].Error != "":
			s.Errors++
		case result.Modules[i].Skipped:
			s.Skipped++
		case result.Modules[i].UpdateAvailable:
			s.UpdatesAvailable++
		}
		if result.Modules[i].Applied {
			s.UpdatesApplied++
		}
	}
	for i := range result.Providers {
		s.TotalChecked++
		switch {
		case result.Providers[i].Error != "":
			s.Errors++
		case result.Providers[i].Skipped:
			s.Skipped++
		case result.Providers[i].UpdateAvailable:
			s.UpdatesAvailable++
		}
		if result.Providers[i].Applied {
			s.UpdatesApplied++
		}
	}
	return s
}

// FindTFFileForModule searches for the .tf file containing a specific module block.
// This is a best-effort search used to populate the File field.
func FindTFFileForModule(modulePath, callName string) string {
	files, err := filepath.Glob(filepath.Join(modulePath, "*.tf"))
	if err != nil {
		return ""
	}
	for _, f := range files {
		if containsModuleBlock(f, callName) {
			return f
		}
	}
	return ""
}

// FindTFFileForProvider searches for the .tf file containing the required_providers block.
func FindTFFileForProvider(modulePath, providerName string) string {
	files, err := filepath.Glob(filepath.Join(modulePath, "*.tf"))
	if err != nil {
		return ""
	}
	for _, f := range files {
		if containsProviderBlock(f, providerName) {
			return f
		}
	}
	return ""
}
