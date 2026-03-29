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
	// Group updates by module path.
	type moduleUpdates struct {
		providers []updateengine.ProviderVersionUpdate
		modules   []updateengine.ModuleVersionUpdate
	}

	// Collect order and grouping.
	groups := make(map[string]*moduleUpdates)
	var order []string

	for i := range result.Providers {
		p := &result.Providers[i]
		if p.Skipped || !p.Updated {
			continue
		}
		g, ok := groups[p.ModulePath]
		if !ok {
			g = &moduleUpdates{}
			groups[p.ModulePath] = g
			order = append(order, p.ModulePath)
		}
		g.providers = append(g.providers, *p)
	}

	for i := range result.Modules {
		m := &result.Modules[i]
		if m.Skipped || !m.Updated {
			continue
		}
		g, ok := groups[m.ModulePath]
		if !ok {
			g = &moduleUpdates{}
			groups[m.ModulePath] = g
			order = append(order, m.ModulePath)
		}
		g.modules = append(g.modules, *m)
	}

	if len(groups) == 0 {
		log.Info("all dependencies are up to date")
		return
	}

	log.WithField("modules", len(groups)).Info("updates available")

	for _, path := range order {
		g := groups[path]
		count := len(g.providers) + len(g.modules)
		log.WithField("updates", count).Info(path)

		log.IncreasePadding()

		for i := range g.providers {
			p := &g.providers[i]
			label := p.ProviderName + " " + p.ProviderSource
			entry := log.WithField("current", formatCurrent(p.Constraint, p.CurrentVersion)).
				WithField("available", p.BumpedVersion)
			if p.LatestVersion != "" && p.LatestVersion != p.BumpedVersion {
				entry = entry.WithField("latest", p.LatestVersion)
			}
			entry.Info(label)
		}

		for i := range g.modules {
			m := &g.modules[i]
			label := m.CallName + " " + m.Source
			entry := log.WithField("current", formatCurrent(m.Constraint, m.CurrentVersion)).
				WithField("available", m.BumpedVersion)
			if m.LatestVersion != "" && m.LatestVersion != m.BumpedVersion {
				entry = entry.WithField("latest", m.LatestVersion)
			}
			entry.Info(label)
		}

		log.DecreasePadding()
	}

	s := result.Summary
	log.WithField("checked", s.TotalChecked).
		WithField("updates", s.UpdatesAvailable).
		WithField("skipped", s.Skipped).
		Info("summary")
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
