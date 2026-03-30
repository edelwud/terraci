package checker

import updateengine "github.com/edelwud/terraci/plugins/update/internal"

type moduleScanResult struct {
	update   updateengine.ModuleVersionUpdate
	analysis versionAnalysis
}

func newModuleScanResult(update updateengine.ModuleVersionUpdate, analysis versionAnalysis) moduleScanResult {
	return moduleScanResult{
		update:   update,
		analysis: analysis,
	}
}

func (r moduleScanResult) outcome(resolveFile func() string) updateengine.ModuleVersionUpdate {
	update := r.update
	if r.analysis.hasCurrent {
		update.CurrentVersion = r.analysis.current.String()
	}
	if !r.analysis.latest.IsZero() {
		update.LatestVersion = r.analysis.latest.String()
	}
	if !r.analysis.bumped.IsZero() {
		return markModuleUpdateAvailable(update, resolveFile(), r.analysis.bumped.String())
	}
	return update
}

type providerScanResult struct {
	update   updateengine.ProviderVersionUpdate
	analysis versionAnalysis
}

func newProviderScanResult(update updateengine.ProviderVersionUpdate, analysis versionAnalysis) providerScanResult {
	return providerScanResult{
		update:   update,
		analysis: analysis,
	}
}

func (r providerScanResult) outcome(resolveFile func() string) updateengine.ProviderVersionUpdate {
	update := r.update
	if r.analysis.hasCurrent {
		update.CurrentVersion = r.analysis.current.String()
	}
	if !r.analysis.latest.IsZero() {
		update.LatestVersion = r.analysis.latest.String()
	}
	if !r.analysis.hasCurrent {
		return skipProviderUpdate(update, "cannot determine current version")
	}
	if !r.analysis.bumped.IsZero() {
		return markProviderUpdateAvailable(update, resolveFile(), r.analysis.bumped.String())
	}
	return update
}
