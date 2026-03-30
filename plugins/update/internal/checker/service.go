package checker

import (
	"context"

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
func (c *Checker) Check(ctx context.Context, modules []*discovery.Module) (*updateengine.UpdateResult, error) {
	return newCheckSession(ctx, c).Run(modules)
}
