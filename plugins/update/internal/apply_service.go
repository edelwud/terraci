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
	if update.Status != StatusUpdateAvailable {
		return
	}
	if update.File == "" {
		update.Status = StatusError
		update.Issue = "failed to locate Terraform file for module update"
		log.WithField("module", update.ModulePath).Warn("failed to locate Terraform file for module update")
		return
	}

	newConstraint := BumpConstraint(update.Constraint, parseVersionOrZero(update.BumpedVersion))
	if err := tfwrite.WriteModuleVersion(update.File, update.CallName, newConstraint); err != nil {
		log.WithField("module", update.ModulePath).WithError(err).Warn("failed to write module version")
		update.Status = StatusError
		update.Issue = fmt.Sprintf("write module version: %v", err)
		return
	}

	update.Status = StatusApplied
	update.Issue = ""
}

func (s *ApplyService) applyProviderUpdate(update *ProviderVersionUpdate) {
	if update.Status != StatusUpdateAvailable {
		return
	}
	if update.File == "" {
		update.Status = StatusError
		update.Issue = "failed to locate Terraform file for provider update"
		log.WithField("provider", update.ProviderSource).Warn("failed to locate Terraform file for provider update")
		return
	}

	newConstraint := BumpConstraint(update.Constraint, parseVersionOrZero(update.BumpedVersion))
	if err := tfwrite.WriteProviderVersion(update.File, update.ProviderName, newConstraint); err != nil {
		log.WithField("provider", update.ProviderSource).WithError(err).Warn("failed to write provider version")
		update.Status = StatusError
		update.Issue = fmt.Sprintf("write provider version: %v", err)
		return
	}

	update.Status = StatusApplied
	update.Issue = ""
}

func parseVersionOrZero(s string) Version {
	v, err := ParseVersion(s)
	if err != nil {
		return Version{}
	}
	return v
}
