package tfupdateengine

import (
	"fmt"
	"strings"

	"github.com/caarlos0/log"

	"github.com/edelwud/terraci/plugins/tfupdate/internal/domain"
	"github.com/edelwud/terraci/plugins/tfupdate/internal/tfwrite"
)

type dependencyMutationApplier struct {
	pin bool
}

func (a dependencyMutationApplier) ApplyModule(update *domain.ModuleVersionUpdate) {
	if !update.IsApplyPending() {
		return
	}

	newConstraint, ok := buildAppliedConstraint(update.BumpedVersion, update.Constraint(), a.pin)
	if !ok {
		markModuleApplyError(update, "failed to build bumped module constraint", nil)
		return
	}

	if err := tfwrite.WriteModuleVersion(update.File, update.CallName(), newConstraint); err != nil {
		markModuleApplyError(update, fmt.Sprintf("write module version: %v", err), err)
		return
	}

	*update = update.MarkApplied()
}

func (a dependencyMutationApplier) ApplyProvider(update *domain.ProviderVersionUpdate) {
	if !update.IsApplyPending() {
		return
	}
	if strings.TrimSpace(update.File) == "" {
		return
	}

	newConstraint, ok := buildAppliedConstraint(update.BumpedVersion, update.Constraint(), a.pin)
	if !ok {
		markProviderApplyError(update, "failed to build bumped provider constraint", nil)
		return
	}

	if err := tfwrite.WriteProviderVersion(update.File, update.ProviderName(), newConstraint); err != nil {
		markProviderApplyError(update, fmt.Sprintf("write provider version: %v", err), err)
		return
	}

	*update = update.MarkApplied()
}

func markModuleApplyError(update *domain.ModuleVersionUpdate, issue string, err error) {
	*update = update.MarkError(issue)
	entry := log.WithField("module", update.ModulePath())
	if err != nil {
		entry = entry.WithError(err)
	}
	entry.Warn(issue)
}

func markProviderApplyError(update *domain.ProviderVersionUpdate, issue string, err error) {
	*update = update.MarkError(issue)
	entry := log.WithField("provider", update.ProviderSource())
	if err != nil {
		entry = entry.WithError(err)
	}
	entry.Warn(issue)
}
