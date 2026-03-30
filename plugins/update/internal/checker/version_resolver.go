package checker

import (
	"sort"

	updateengine "github.com/edelwud/terraci/plugins/update/internal"
)

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
	return analyzeLatestStable(sortVersionsDesc(versions))
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

	sorted := sortVersionsDesc(versions)
	analysis.latest = analyzeLatestStable(sorted)
	if analysis.hasCurrent {
		analysis.bumped = findBumpedVersion(sorted, current, bump)
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

	sorted := sortVersionsDesc(versions)
	analysis.latest = analyzeLatestStable(sorted)

	if !analysis.hasCurrent && constraint != "" {
		constraints, err := updateengine.ParseConstraints(constraint)
		if err == nil {
			analysis.current, analysis.hasCurrent = findLatestAllowed(sorted, constraints)
		}
	}

	if analysis.hasCurrent {
		analysis.bumped = findBumpedVersion(sorted, analysis.current, bump)
	}

	return analysis
}

func sortVersionsDesc(versions []updateengine.Version) []updateengine.Version {
	sorted := make([]updateengine.Version, len(versions))
	copy(sorted, versions)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Compare(sorted[j]) > 0
	})
	return sorted
}

func analyzeLatestStable(sorted []updateengine.Version) updateengine.Version {
	for _, version := range sorted {
		if version.Prerelease == "" {
			return version
		}
	}
	return updateengine.Version{}
}

func findLatestAllowed(
	sorted []updateengine.Version,
	constraints []updateengine.Constraint,
) (updateengine.Version, bool) {
	for _, version := range sorted {
		if version.Prerelease != "" {
			continue
		}
		if updateengine.SatisfiesAll(version, constraints) {
			return version, true
		}
	}
	return updateengine.Version{}, false
}

func findBumpedVersion(
	sorted []updateengine.Version,
	current updateengine.Version,
	bump string,
) updateengine.Version {
	for _, version := range sorted {
		if version.Prerelease != "" || version.Compare(current) <= 0 {
			continue
		}
		switch bump {
		case updateengine.BumpPatch:
			if version.Major == current.Major && version.Minor == current.Minor {
				return version
			}
		case updateengine.BumpMinor:
			if version.Major == current.Major {
				return version
			}
		case updateengine.BumpMajor:
			return version
		}
	}
	return updateengine.Version{}
}
