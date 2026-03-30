package update

import (
	"encoding/json"
	"io"

	"github.com/edelwud/terraci/pkg/log"
	updateengine "github.com/edelwud/terraci/plugins/update/internal"
)

func outputResult(w io.Writer, format string, result *updateengine.UpdateResult) error {
	if format == "json" {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	}

	outputLog(result)
	return nil
}

func outputLog(result *updateengine.UpdateResult) {
	groups, order := collectModuleUpdates(result)
	if len(groups) == 0 {
		logNoUpdatesSummary(result.Summary)
		return
	}

	log.WithField("modules", len(groups)).Info("updates available")
	for _, path := range order {
		logModuleUpdates(path, groups[path])
	}

	logSummary(result.Summary)
}

type moduleUpdates struct {
	providers []*updateengine.ProviderVersionUpdate
	modules   []*updateengine.ModuleVersionUpdate
}

func collectModuleUpdates(result *updateengine.UpdateResult) (groups map[string]*moduleUpdates, order []string) {
	groups = make(map[string]*moduleUpdates)

	for i := range result.Providers {
		p := &result.Providers[i]
		if !p.IncludedInUpdateLogs() {
			continue
		}
		modulePath := p.ModulePath()
		groups, order = ensureModuleGroup(groups, order, modulePath)
		groups[modulePath].providers = append(groups[modulePath].providers, p)
	}

	for i := range result.Modules {
		m := &result.Modules[i]
		if !m.IncludedInUpdateLogs() {
			continue
		}
		modulePath := m.ModulePath()
		groups, order = ensureModuleGroup(groups, order, modulePath)
		groups[modulePath].modules = append(groups[modulePath].modules, m)
	}

	return groups, order
}

func ensureModuleGroup(groups map[string]*moduleUpdates, order []string, modulePath string) (updatedGroups map[string]*moduleUpdates, updatedOrder []string) {
	if _, ok := groups[modulePath]; ok {
		return groups, order
	}
	groups[modulePath] = &moduleUpdates{}
	order = append(order, modulePath)
	return groups, order
}

func logModuleUpdates(path string, updates *moduleUpdates) {
	count := len(updates.providers) + len(updates.modules)
	log.WithField("updates", count).Info(path)
	log.IncreasePadding()
	for _, update := range updates.providers {
		logProviderUpdate(update)
	}
	for _, update := range updates.modules {
		logModuleUpdate(update)
	}
	log.DecreasePadding()
}

func logProviderUpdate(update *updateengine.ProviderVersionUpdate) {
	label := update.ProviderName() + " " + update.ProviderSource()
	entry := log.WithField("current", formatCurrent(update.Constraint(), update.CurrentVersion)).
		WithField("available", update.BumpedVersion)
	if update.Status == updateengine.StatusApplied {
		entry = entry.WithField("status", string(update.Status))
	}
	if update.LatestVersion != "" && update.LatestVersion != update.BumpedVersion {
		entry = entry.WithField("latest", update.LatestVersion)
	}
	entry.Info(label)
}

func logModuleUpdate(update *updateengine.ModuleVersionUpdate) {
	label := update.CallName() + " " + update.Source()
	entry := log.WithField("current", formatCurrent(update.Constraint(), update.CurrentVersion)).
		WithField("available", update.BumpedVersion)
	if update.Status == updateengine.StatusApplied {
		entry = entry.WithField("status", string(update.Status))
	}
	if update.LatestVersion != "" && update.LatestVersion != update.BumpedVersion {
		entry = entry.WithField("latest", update.LatestVersion)
	}
	entry.Info(label)
}

func logNoUpdatesSummary(summary updateengine.UpdateSummary) {
	log.Info("summary")
	log.IncreasePadding()
	log.WithField("checked", summary.TotalChecked).Info("checked")
	if summary.Skipped > 0 {
		log.WithField("count", summary.Skipped).Warn("skipped")
	}
	log.DecreasePadding()
	log.Info("all dependencies are up to date")
}

func logSummary(summary updateengine.UpdateSummary) {
	log.Info("summary")
	log.IncreasePadding()
	log.WithField("count", summary.TotalChecked).Info("checked")
	log.WithField("count", summary.UpdatesAvailable).Warn("updates available")
	if summary.UpdatesApplied > 0 {
		log.WithField("count", summary.UpdatesApplied).Info("updates applied")
	}
	if summary.Errors > 0 {
		log.WithField("count", summary.Errors).Warn("errors")
	}
	if summary.Skipped > 0 {
		log.WithField("count", summary.Skipped).Warn("skipped")
	}
	log.DecreasePadding()
}

// formatCurrent renders the current version field.
// When constraint differs from resolved version, shows "~> 5.0 (5.84.0)".
func formatCurrent(constraint, resolved string) string {
	if resolved == "" || constraint == resolved {
		return constraint
	}
	if constraint == "" {
		return resolved
	}
	return constraint + " (" + resolved + ")"
}
