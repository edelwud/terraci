package planner

import (
	"sort"

	tfupdateengine "github.com/edelwud/terraci/plugins/tfupdate/internal"
	"github.com/edelwud/terraci/plugins/tfupdate/internal/versionkit"
)

type versionAnalysis struct {
	current    versionkit.Version
	latest     versionkit.Version
	bumped     versionkit.Version
	hasCurrent bool
}

func parseVersionList(strs []string) []versionkit.Version {
	versions := make([]versionkit.Version, 0, len(strs))
	for _, s := range strs {
		v, err := versionkit.ParseVersion(s)
		if err != nil {
			continue
		}
		versions = append(versions, v)
	}
	return versions
}

func latestStable(versions []versionkit.Version) versionkit.Version {
	return analyzeLatestStable(versions)
}

func versionFromConstraint(s string) versionkit.Version {
	constraints, err := versionkit.ParseConstraints(s)
	if err != nil || len(constraints) == 0 {
		v, parseErr := versionkit.ParseVersion(s)
		if parseErr != nil {
			return versionkit.Version{}
		}
		return v
	}
	return constraints[0].Version
}

func analyzeProviderVersions(constraint, currentVersion string, versions []versionkit.Version, bump string) versionAnalysis {
	analysis := versionAnalysis{}
	if currentVersion != "" {
		if version, err := versionkit.ParseVersion(currentVersion); err == nil {
			analysis.current = version
			analysis.hasCurrent = true
		}
	}

	analysis.latest = analyzeLatestStable(versions)

	if !analysis.hasCurrent && constraint != "" {
		constraints, err := versionkit.ParseConstraints(constraint)
		if err == nil {
			analysis.current, analysis.hasCurrent = findLatestAllowed(versions, constraints)
		}
	}

	if analysis.hasCurrent {
		analysis.bumped = findBumpedVersion(versions, analysis.current, bump)
	}
	return analysis
}

func analyzeLatestStable(versions []versionkit.Version) versionkit.Version {
	var best versionkit.Version
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

func findLatestAllowed(versions []versionkit.Version, constraints []versionkit.Constraint) (versionkit.Version, bool) {
	var best versionkit.Version
	found := false
	for _, version := range versions {
		if version.Prerelease != "" {
			continue
		}
		if !versionkit.SatisfiesAll(version, constraints) {
			continue
		}
		if !found || version.Compare(best) > 0 {
			best = version
			found = true
		}
	}
	return best, found
}

func findBumpedVersion(versions []versionkit.Version, current versionkit.Version, bump string) versionkit.Version {
	var best versionkit.Version
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

func sortVersionsDesc(versions []versionkit.Version) []versionkit.Version {
	sorted := append([]versionkit.Version(nil), versions...)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Compare(sorted[j]) > 0
	})
	return sorted
}

func sortBumpCandidates(versions []versionkit.Version, current versionkit.Version, bump string) []versionkit.Version {
	sorted := sortVersionsDesc(versions)
	result := make([]versionkit.Version, 0, len(sorted))
	for _, version := range sorted {
		if version.Compare(current) <= 0 || !withinBump(current, version, bump) {
			continue
		}
		result = append(result, version)
	}
	return result
}

func withinBump(current, candidate versionkit.Version, bump string) bool {
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
