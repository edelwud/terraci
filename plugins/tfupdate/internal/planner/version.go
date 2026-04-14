package planner

import (
	"sort"

	tfupdateengine "github.com/edelwud/terraci/plugins/tfupdate/internal"
)

type versionAnalysis struct {
	current    tfupdateengine.Version
	latest     tfupdateengine.Version
	bumped     tfupdateengine.Version
	hasCurrent bool
}

func parseVersionList(strs []string) []tfupdateengine.Version {
	versions := make([]tfupdateengine.Version, 0, len(strs))
	for _, s := range strs {
		v, err := tfupdateengine.ParseVersion(s)
		if err != nil {
			continue
		}
		versions = append(versions, v)
	}
	return versions
}

func latestStable(versions []tfupdateengine.Version) tfupdateengine.Version {
	return analyzeLatestStable(versions)
}

func versionFromConstraint(s string) tfupdateengine.Version {
	constraints, err := tfupdateengine.ParseConstraints(s)
	if err != nil || len(constraints) == 0 {
		v, parseErr := tfupdateengine.ParseVersion(s)
		if parseErr != nil {
			return tfupdateengine.Version{}
		}
		return v
	}
	return constraints[0].Version
}

func analyzeProviderVersions(constraint, currentVersion string, versions []tfupdateengine.Version, bump string) versionAnalysis {
	analysis := versionAnalysis{}
	if currentVersion != "" {
		if version, err := tfupdateengine.ParseVersion(currentVersion); err == nil {
			analysis.current = version
			analysis.hasCurrent = true
		}
	}

	analysis.latest = analyzeLatestStable(versions)

	if !analysis.hasCurrent && constraint != "" {
		constraints, err := tfupdateengine.ParseConstraints(constraint)
		if err == nil {
			analysis.current, analysis.hasCurrent = findLatestAllowed(versions, constraints)
		}
	}

	if analysis.hasCurrent {
		analysis.bumped = findBumpedVersion(versions, analysis.current, bump)
	}
	return analysis
}

func analyzeLatestStable(versions []tfupdateengine.Version) tfupdateengine.Version {
	var best tfupdateengine.Version
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

func findLatestAllowed(versions []tfupdateengine.Version, constraints []tfupdateengine.Constraint) (tfupdateengine.Version, bool) {
	var best tfupdateengine.Version
	found := false
	for _, version := range versions {
		if version.Prerelease != "" {
			continue
		}
		if !tfupdateengine.SatisfiesAll(version, constraints) {
			continue
		}
		if !found || version.Compare(best) > 0 {
			best = version
			found = true
		}
	}
	return best, found
}

func findBumpedVersion(versions []tfupdateengine.Version, current tfupdateengine.Version, bump string) tfupdateengine.Version {
	var best tfupdateengine.Version
	for _, version := range versions {
		if version.Prerelease != "" || version.Compare(current) <= 0 {
			continue
		}
		switch bump {
		case tfupdateengine.BumpPatch:
			if version.Major == current.Major && version.Minor == current.Minor && version.Compare(best) > 0 {
				best = version
			}
		case tfupdateengine.BumpMinor:
			if version.Major == current.Major && version.Compare(best) > 0 {
				best = version
			}
		case tfupdateengine.BumpMajor:
			if version.Compare(best) > 0 {
				best = version
			}
		}
	}
	return best
}

func sortVersionsDesc(versions []tfupdateengine.Version) []tfupdateengine.Version {
	sorted := append([]tfupdateengine.Version(nil), versions...)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Compare(sorted[j]) > 0
	})
	return sorted
}

func sortBumpCandidates(versions []tfupdateengine.Version, current tfupdateengine.Version, bump string) []tfupdateengine.Version {
	sorted := sortVersionsDesc(versions)
	result := make([]tfupdateengine.Version, 0, len(sorted))
	for _, version := range sorted {
		if version.Compare(current) <= 0 || !withinBump(current, version, bump) {
			continue
		}
		result = append(result, version)
	}
	return result
}

func withinBump(current, candidate tfupdateengine.Version, bump string) bool {
	switch bump {
	case tfupdateengine.BumpPatch:
		return candidate.Major == current.Major && candidate.Minor == current.Minor
	case tfupdateengine.BumpMinor:
		return candidate.Major == current.Major
	case tfupdateengine.BumpMajor:
		return true
	default:
		return false
	}
}
