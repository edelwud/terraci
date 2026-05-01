package planner

import (
	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/parser"
	"github.com/edelwud/terraci/plugins/tfupdate/internal/domain"
)

func (s *Solver) SolveModule(mod *discovery.Module, parsed *parser.ParsedModule) (*domain.PlanResult, error) {
	scanCtx := newModuleScanContext(mod, parsed)
	modulePlans, providerPlans := s.buildPlans(scanCtx)
	selected, providerSelections, compatible := s.findCompatiblePlan(modulePlans, providerPlans)
	lockSync, err := s.buildLockSyncPlan(scanCtx, providerPlans, selected, providerSelections, compatible)
	if err != nil {
		return nil, err
	}

	return &domain.PlanResult{
		ModulePath: mod.RelativePath,
		Modules:    s.materializeTargetModuleResolutions(scanCtx, modulePlans, selected, compatible),
		Providers:  s.materializeTargetProviderResolutions(scanCtx, providerPlans, providerSelections, compatible),
		LockSync:   lockSync,
	}, nil
}
