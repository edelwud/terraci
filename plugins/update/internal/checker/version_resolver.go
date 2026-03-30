package checker

import updateengine "github.com/edelwud/terraci/plugins/update/internal"

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
	var best updateengine.Version
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
// E.g. "~> 5.0" -> 5.0.0, ">= 1.2.3" -> 1.2.3, "5.0" -> 5.0.0.
func versionFromConstraint(s string) updateengine.Version {
	constraints, err := updateengine.ParseConstraints(s)
	if err != nil || len(constraints) == 0 {
		v, _ := updateengine.ParseVersion(s) //nolint:errcheck // best-effort
		return v
	}
	return constraints[0].Version
}
