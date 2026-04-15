package tfupdateengine

import (
	"context"
	"fmt"
	"strings"

	"github.com/caarlos0/log"

	"github.com/edelwud/terraci/plugins/tfupdate/internal/domain"
	"github.com/edelwud/terraci/plugins/tfupdate/internal/lockfile"
)

type lockSyncApplier struct {
	ctx    context.Context
	syncer lockfile.Syncer
}

func (a lockSyncApplier) Apply(result *UpdateResult, plans []domain.LockSyncPlan) {
	for _, plan := range plans {
		for _, provider := range plan.Providers {
			if a.ctx != nil && a.ctx.Err() != nil {
				return
			}
			if provider.ProviderSource == "" || provider.Version == "" || provider.TerraformFile == "" {
				continue
			}

			log.WithField("provider", provider.ProviderSource).
				WithField("version", provider.Version).
				Info("syncing provider lock file")

			if err := a.syncer.SyncProvider(a.ctx, lockfile.ProviderLockRequest{
				ProviderSource: provider.ProviderSource,
				Version:        provider.Version,
				Constraint:     provider.Constraint,
				TerraformFile:  provider.TerraformFile,
			}); err != nil {
				markLockSyncError(result, provider, err)
				log.WithField("provider", provider.ProviderSource).WithError(err).Warn("failed to sync lock file")
				continue
			}
			markLockSyncApplied(result, provider)
		}
	}
}

func markLockSyncError(result *UpdateResult, provider domain.LockProviderSync, err error) {
	if result == nil {
		return
	}

	issue := fmt.Sprintf("update provider lock file: %v", err)
	for i := range result.Providers {
		update := &result.Providers[i]
		if update.ProviderSource() != provider.ProviderSource || update.File != provider.TerraformFile {
			continue
		}
		*update = update.MarkError(issue)
		return
	}
}

func markLockSyncApplied(result *UpdateResult, provider domain.LockProviderSync) {
	if result == nil {
		return
	}

	for i := range result.Providers {
		update := &result.Providers[i]
		if update.ProviderSource() != provider.ProviderSource {
			continue
		}
		if strings.TrimSpace(update.File) == "" || update.File == provider.TerraformFile {
			*update = update.MarkApplied()
			return
		}
	}
}
