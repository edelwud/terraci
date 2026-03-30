package checker

import (
	"context"
	"fmt"
	"runtime"

	"github.com/caarlos0/log"
	"golang.org/x/sync/errgroup"

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
	results, err := s.checkModules(modules)
	if err != nil {
		return nil, fmt.Errorf("walk modules: %w", err)
	}

	s.mergeModuleResults(results)

	result := s.builder.Result()
	if s.checker.write {
		updateengine.NewApplyService().Apply(result)
	}

	return s.builder.Build(), nil
}

type moduleCheckResult struct {
	moduleUpdates   []updateengine.ModuleVersionUpdate
	providerUpdates []updateengine.ProviderVersionUpdate
	parseError      bool
}

func (s *checkSession) checkModules(modules []*discovery.Module) ([]moduleCheckResult, error) {
	results := make([]moduleCheckResult, len(modules))

	var group errgroup.Group
	group.SetLimit(checkConcurrencyLimit())

	for i := range modules {
		group.Go(func() error {
			if err := s.ctx.Err(); err != nil {
				return err
			}

			result, err := s.checkModule(modules[i])
			if err != nil {
				return err
			}

			results[i] = result
			return nil
		})
	}

	if err := group.Wait(); err != nil {
		return nil, err
	}

	return results, nil
}

func (s *checkSession) checkModule(
	mod *discovery.Module,
) (moduleCheckResult, error) {
	parsed, err := s.checker.parser.ParseModule(s.ctx, mod.Path)
	if err != nil {
		return s.handleParseError(mod, err)
	}

	return s.handleParsedModule(mod, parsed), nil
}

func (s *checkSession) handleParsedModule(
	mod *discovery.Module,
	parsed *parser.ParsedModule,
) moduleCheckResult {
	scanCtx := s.newModuleScanContext(mod, parsed)
	result := moduleCheckResult{}

	if s.checker.config.ShouldCheckProviders() {
		result.providerUpdates = s.collectProviderUpdates(scanCtx)
	}
	if s.checker.config.ShouldCheckModules() {
		result.moduleUpdates = s.collectModuleUpdates(scanCtx)
	}

	return result
}

func (s *checkSession) handleParseError(
	mod *discovery.Module,
	err error,
) (moduleCheckResult, error) {
	s.recordParseError(mod, err)
	return moduleCheckResult{parseError: true}, nil
}

func (s *checkSession) recordParseError(mod *discovery.Module, err error) {
	log.WithField("module", mod.RelativePath).WithError(err).Warn("failed to parse module")
	s.builder.RecordError()
}

func (s *checkSession) mergeModuleResults(results []moduleCheckResult) {
	for i := range results {
		result := &results[i]
		if result.parseError {
			continue
		}
		for j := range result.providerUpdates {
			s.builder.AddProviderUpdate(result.providerUpdates[j])
		}
		for j := range result.moduleUpdates {
			s.builder.AddModuleUpdate(result.moduleUpdates[j])
		}
	}
}

func checkConcurrencyLimit() int {
	const maxConcurrency = 8

	if cpu := runtime.GOMAXPROCS(0); cpu > 0 && cpu < maxConcurrency {
		return cpu
	}

	return maxConcurrency
}
