package checker

import updateengine "github.com/edelwud/terraci/plugins/update/internal"

type versionAnalysis struct {
	current    updateengine.Version
	latest     updateengine.Version
	bumped     updateengine.Version
	hasCurrent bool
}

// parseVersionList parses semver strings and ignores unparseable entries.
func parseVersionList(strs []string) []updateengine.Version {
	versions := make([]updateengine.Version, 0, len(strs))
	for _, s := range strs {
		v, err := updateengine.ParseVersion(s)
		if err != nil {
			continue
		}
		versions = append(versions, v)
	}
	return versions
}

func latestStable(versions []updateengine.Version) updateengine.Version {
	return analyzeLatestStable(versions)
}

// versionFromConstraint extracts the base version from a constraint string.
// E.g. "~> 5.0" -> 5.0.0, ">= 1.2.3" -> 1.2.3, "5.0" -> 5.0.0.
func versionFromConstraint(s string) updateengine.Version {
	constraints, err := updateengine.ParseConstraints(s)
	if err != nil || len(constraints) == 0 {
		v, _ := updateengine.ParseVersion(s) //nolint:errcheck // best-effort
		return v
	}
	return constraints[0].Version
}

func analyzeModuleVersions(
	constraint string,
	versions []updateengine.Version,
	bump string,
) versionAnalysis {
	current := versionFromConstraint(constraint)
	analysis := versionAnalysis{
		current:    current,
		hasCurrent: !current.IsZero(),
	}

	analysis.latest = analyzeLatestStable(versions)
	if analysis.hasCurrent {
		analysis.bumped = findBumpedVersion(versions, current, bump)
	}

	return analysis
}

func analyzeProviderVersions(
	constraint string,
	currentVersion string,
	versions []updateengine.Version,
	bump string,
) versionAnalysis {
	analysis := versionAnalysis{}
	if currentVersion != "" {
		if version, err := updateengine.ParseVersion(currentVersion); err == nil {
			analysis.current = version
			analysis.hasCurrent = true
		}
	}

	analysis.latest = analyzeLatestStable(versions)

	if !analysis.hasCurrent && constraint != "" {
		constraints, err := updateengine.ParseConstraints(constraint)
		if err == nil {
			analysis.current, analysis.hasCurrent = findLatestAllowed(versions, constraints)
		}
	}

	if analysis.hasCurrent {
		analysis.bumped = findBumpedVersion(versions, analysis.current, bump)
	}

	return analysis
}

func analyzeLatestStable(versions []updateengine.Version) updateengine.Version {
	var best updateengine.Version
	for _, version := range versions {
		if version.Prerelease != "" {
			continue
		}
		if version.Compare(best) > 0 {
			best = version
		}
	}
	return best
}

func findLatestAllowed(
	versions []updateengine.Version,
	constraints []updateengine.Constraint,
) (updateengine.Version, bool) {
	var best updateengine.Version
	found := false
	for _, version := range versions {
		if version.Prerelease != "" {
			continue
		}
		if !updateengine.SatisfiesAll(version, constraints) {
			continue
		}
		if !found || version.Compare(best) > 0 {
			best = version
			found = true
		}
	}
	return best, found
}

func findBumpedVersion(
	versions []updateengine.Version,
	current updateengine.Version,
	bump string,
) updateengine.Version {
	var best updateengine.Version
	for _, version := range versions {
		if version.Prerelease != "" || version.Compare(current) <= 0 {
			continue
		}
		switch bump {
		case updateengine.BumpPatch:
			if version.Major == current.Major && version.Minor == current.Minor && version.Compare(best) > 0 {
				best = version
			}
		case updateengine.BumpMinor:
			if version.Major == current.Major && version.Compare(best) > 0 {
				best = version
			}
		case updateengine.BumpMajor:
			if version.Compare(best) > 0 {
				best = version
			}
		}
	}
	return best
}
