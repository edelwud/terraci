package updateengine

import (
	"fmt"

	"github.com/caarlos0/log"

	"github.com/edelwud/terraci/plugins/update/internal/tfwrite"
)

// ApplyService applies discovered dependency updates to Terraform files.
type ApplyService struct{}

// NewApplyService constructs a stateless update apply service.
func NewApplyService() *ApplyService {
	return &ApplyService{}
}

// Apply mutates the result items in place with apply outcomes.
func (s *ApplyService) Apply(result *UpdateResult) {
	for i := range result.Modules {
		s.applyModuleUpdate(&result.Modules[i])
	}

	for i := range result.Providers {
		s.applyProviderUpdate(&result.Providers[i])
	}
}

func (s *ApplyService) applyModuleUpdate(update *ModuleVersionUpdate) {
	if !update.IsApplyPending() {
		return
	}

	newConstraint, ok := buildAppliedConstraint(update.BumpedVersion, update.Constraint())
	if !ok {
		s.markModuleApplyError(update, "failed to build bumped module constraint", nil)
		return
	}

	if err := tfwrite.WriteModuleVersion(update.File, update.CallName(), newConstraint); err != nil {
		s.markModuleApplyError(update, fmt.Sprintf("write module version: %v", err), err)
		return
	}

	*update = update.MarkApplied()
}

func (s *ApplyService) applyProviderUpdate(update *ProviderVersionUpdate) {
	if !update.IsApplyPending() {
		return
	}

	newConstraint, ok := buildAppliedConstraint(update.BumpedVersion, update.Constraint())
	if !ok {
		s.markProviderApplyError(update, "failed to build bumped provider constraint", nil)
		return
	}

	if err := tfwrite.WriteProviderVersion(update.File, update.ProviderName(), newConstraint); err != nil {
		s.markProviderApplyError(update, fmt.Sprintf("write provider version: %v", err), err)
		return
	}

	*update = update.MarkApplied()
}

func parseVersionOrZero(s string) Version {
	v, err := ParseVersion(s)
	if err != nil {
		return Version{}
	}
	return v
}

func buildAppliedConstraint(bumpedVersion, originalConstraint string) (string, bool) {
	version := parseVersionOrZero(bumpedVersion)
	if version.IsZero() {
		return "", false
	}
	return BumpConstraint(originalConstraint, version), true
}

func (s *ApplyService) markModuleApplyError(update *ModuleVersionUpdate, issue string, err error) {
	*update = update.MarkError(issue)
	entry := log.WithField("module", update.ModulePath())
	if err != nil {
		entry = entry.WithError(err)
	}
	entry.Warn(issue)
}

func (s *ApplyService) markProviderApplyError(update *ProviderVersionUpdate, issue string, err error) {
	*update = update.MarkError(issue)
	entry := log.WithField("provider", update.ProviderSource())
	if err != nil {
		entry = entry.WithError(err)
	}
	entry.Warn(issue)
}
