package tfupdateengine

import "github.com/edelwud/terraci/plugins/tfupdate/internal/versionkit"

func parseVersionOrZero(s string) versionkit.Version {
	v, err := versionkit.ParseVersion(s)
	if err != nil {
		return versionkit.Version{}
	}
	return v
}

func buildAppliedConstraint(bumpedVersion, originalConstraint string, pin bool) (string, bool) {
	version := parseVersionOrZero(bumpedVersion)
	if version.IsZero() {
		return "", false
	}
	if pin {
		return version.String(), true
	}
	return versionkit.BumpConstraint(originalConstraint, version), true
}

func isExactConstraint(constraint, version string) bool {
	cs, err := versionkit.ParseConstraints(constraint)
	if err != nil || len(cs) != 1 {
		return false
	}
	return cs[0].Op == versionkit.OpEqual && cs[0].Version.String() == version
}
