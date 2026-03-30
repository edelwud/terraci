package checker

import (
	"context"
	"fmt"

	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/parser"
	updateengine "github.com/edelwud/terraci/plugins/update/internal"
)

const skipReasonIgnored = "ignored by config"

// Checker performs version checks across Terraform modules.
type Checker struct {
	config   *updateengine.UpdateConfig
	parser   *parser.Parser
	registry updateengine.RegistryClient
	write    bool
}

// NewChecker creates a new dependency version checker.
func NewChecker(
	cfg *updateengine.UpdateConfig,
	moduleParser *parser.Parser,
	registry updateengine.RegistryClient,
	write bool,
) *Checker {
	return &Checker{
		config:   cfg,
		parser:   moduleParser,
		registry: registry,
		write:    write,
	}
}

// Check performs version checks on all provided modules.
func (s *Checker) Check(ctx context.Context, modules []*discovery.Module) (*updateengine.UpdateResult, error) {
	result := updateengine.NewUpdateResult()
	err := walkModules(
		ctx,
		s.parser,
		modules,
		func(ctx context.Context, mod *discovery.Module, parsed *parser.ParsedModule) error {
			if s.config.ShouldCheckProviders() {
				s.checkProviderUpdates(ctx, mod, parsed, result)
			}
			if s.config.ShouldCheckModules() {
				s.checkModuleUpdates(ctx, mod, parsed, result)
			}
			return nil
		},
		func(_ *discovery.Module, _ error) error {
			result.RecordError()
			return nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf("walk modules: %w", err)
	}

	if s.write {
		updateengine.NewApplyService().Apply(result)
	}

	result.Summary = updateengine.BuildUpdateSummary(result)
	return result, nil
}
