package planner

import (
	"github.com/edelwud/terraci/pkg/parser"
	tfupdateengine "github.com/edelwud/terraci/plugins/tfupdate/internal"
	"github.com/edelwud/terraci/plugins/tfupdate/internal/domain"
	"github.com/edelwud/terraci/plugins/tfupdate/internal/sourceaddr"
)

func resolveRegistryProviderAddress(source string, lockedProvider *parser.LockedProvider, cfg *tfupdateengine.UpdateConfig) (sourceaddr.ProviderAddress, error) {
	address, err := sourceaddr.ParseProviderAddress(source)
	if err != nil {
		return sourceaddr.ProviderAddress{}, err
	}
	address = address.WithHostname(cfg.ProviderRegistryHost(address.ShortSource()))
	namespace, typeName, parseErr := sourceaddr.ParseProviderSource(source)
	if parseErr != nil || lockedProvider == nil || lockedProvider.Source == "" {
		return address, nil //nolint:nilerr // parse failure means short source is unusable; fall back to default address
	}
	lockedAddress, lockedErr := sourceaddr.ParseProviderAddress(lockedProvider.Source)
	if lockedErr != nil {
		return address, nil //nolint:nilerr // locked source parse failure is non-fatal; fall back to resolved address
	}
	if lockedAddress.Namespace != namespace || lockedAddress.Type != typeName {
		return address, nil
	}
	return lockedAddress, nil
}

func withLockedProviderState(update domain.ProviderVersionUpdate, lockedProvider *parser.LockedProvider) domain.ProviderVersionUpdate {
	if lockedProvider == nil {
		return update
	}
	if lockedProvider.Version != "" {
		update.CurrentVersion = lockedProvider.Version
	}
	if update.Constraint() == "" && lockedProvider.Constraints != "" {
		update.Dependency.Constraint = lockedProvider.Constraints
	}
	return update
}
