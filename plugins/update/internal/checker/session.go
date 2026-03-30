package checker

import (
	"context"
	"fmt"

	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/parser"
	updateengine "github.com/edelwud/terraci/plugins/update/internal"
)

// checkSession coordinates a single end-to-end dependency check pass.
type checkSession struct {
	ctx     context.Context
	checker *Checker
	builder *updateengine.UpdateResultBuilder
}

func newCheckSession(ctx context.Context, checker *Checker) *checkSession {
	return &checkSession{
		ctx:     ctx,
		checker: checker,
		builder: updateengine.NewUpdateResultBuilder(),
	}
}

func (s *checkSession) Run(modules []*discovery.Module) (*updateengine.UpdateResult, error) {
	err := walkModules(
		s.ctx,
		s.checker.parser,
		modules,
		s.handleParsedModule,
		s.handleParseError,
	)
	if err != nil {
		return nil, fmt.Errorf("walk modules: %w", err)
	}

	result := s.builder.Result()
	if s.checker.write {
		updateengine.NewApplyService().Apply(result)
	}

	return s.builder.Build(), nil
}

func (s *checkSession) handleParsedModule(
	_ context.Context,
	mod *discovery.Module,
	parsed *parser.ParsedModule,
) error {
	if s.checker.config.ShouldCheckProviders() {
		s.collectProviderUpdates(mod, parsed)
	}
	if s.checker.config.ShouldCheckModules() {
		s.collectModuleUpdates(mod, parsed)
	}
	return nil
}

func (s *checkSession) handleParseError(_ *discovery.Module, _ error) error {
	s.builder.RecordError()
	return nil
}
