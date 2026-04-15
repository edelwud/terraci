package planner

func (s *Solver) buildPlans(scanCtx *moduleScanContext) ([]*modulePlan, []*providerPlan) {
	modulePlans := make([]*modulePlan, 0, len(scanCtx.parsed.ModuleCalls))
	for _, mc := range scanCtx.parsed.ModuleCalls {
		modulePlans = append(modulePlans, s.buildModulePlan(scanCtx, mc))
	}

	providerPlans := make([]*providerPlan, 0, len(scanCtx.parsed.RequiredProviders)+len(scanCtx.parsed.LockedProviders))
	seenProviders := make(map[string]struct{}, len(scanCtx.parsed.RequiredProviders))
	for _, rp := range scanCtx.parsed.RequiredProviders {
		providerPlans = append(providerPlans, s.buildProviderPlan(scanCtx, rp))
		seenProviders[rp.Source] = struct{}{}
	}
	for _, lp := range scanCtx.parsed.LockedProviders {
		short := stripRegistryPrefix(lp.Source)
		if _, ok := seenProviders[short]; ok {
			continue
		}
		if plan := s.buildLockedProviderPlan(scanCtx, lp); plan != nil {
			providerPlans = append(providerPlans, plan)
		}
	}
	return modulePlans, providerPlans
}
