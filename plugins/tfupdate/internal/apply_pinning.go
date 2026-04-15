package tfupdateengine

import (
	"strings"

	"github.com/caarlos0/log"

	"github.com/edelwud/terraci/plugins/tfupdate/internal/domain"
	"github.com/edelwud/terraci/plugins/tfupdate/internal/tfwrite"
)

type dependencyPinner struct{}

func (dependencyPinner) PinProviders(result *UpdateResult) {
	for i := range result.Providers {
		p := &result.Providers[i]
		if p.Status != domain.StatusUpToDate {
			continue
		}
		if strings.TrimSpace(p.File) == "" || p.CurrentVersion == "" {
			continue
		}
		if isExactConstraint(p.Constraint(), p.CurrentVersion) {
			continue
		}

		if err := tfwrite.WriteProviderVersion(p.File, p.ProviderName(), p.CurrentVersion); err != nil {
			log.WithField("provider", p.ProviderSource()).WithError(err).Warn("failed to pin provider version")
			continue
		}

		log.WithField("provider", p.ProviderSource()).
			WithField("version", p.CurrentVersion).
			Info("pinned provider version")
	}
}

func (dependencyPinner) PinModules(result *UpdateResult) {
	for i := range result.Modules {
		m := &result.Modules[i]
		if m.Status != domain.StatusUpToDate {
			continue
		}
		if strings.TrimSpace(m.File) == "" {
			continue
		}

		targetVersion := m.BumpedVersion
		if targetVersion == "" {
			targetVersion = m.CurrentVersion
		}
		if targetVersion == "" || isExactConstraint(m.Constraint(), targetVersion) {
			continue
		}

		if err := tfwrite.WriteModuleVersion(m.File, m.CallName(), targetVersion); err != nil {
			log.WithField("module", m.CallName()).WithError(err).Warn("failed to pin module version")
			continue
		}

		log.WithField("module", m.CallName()).
			WithField("version", targetVersion).
			Info("pinned module version")
	}
}
