package costengine

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/caarlos0/log"
	"golang.org/x/sync/errgroup"

	"github.com/edelwud/terraci/internal/terraform/plan"
	"github.com/edelwud/terraci/plugins/cost/internal/cloud"
	"github.com/edelwud/terraci/plugins/cost/internal/cloud/awskit"
	"github.com/edelwud/terraci/plugins/cost/internal/handler"
	"github.com/edelwud/terraci/plugins/cost/internal/pricing"
)

// DefaultRegion is used when no region is specified.
const DefaultRegion = "us-east-1"

// Estimator calculates cost estimates for terraform plans.
type Estimator struct {
	resolver *CostResolver
	runtimes map[string]*ProviderRuntime
}

// NewEstimator creates a new cost estimator with the given pricing fetcher.
func NewEstimator(cacheDir string, cacheTTL time.Duration, fetcher pricing.PriceFetcher) *Estimator {
	providers := cloud.Providers()
	registry := newDefaultRegistry(providers)
	runtimes := make(map[string]*ProviderRuntime, len(providers))

	for _, cp := range providers {
		def := cp.Definition()
		runtimeFetcher := def.FetcherFactory()
		if len(providers) == 1 && fetcher != nil {
			runtimeFetcher = fetcher
		}
		runtimes[def.Manifest.ID] = &ProviderRuntime{
			Definition: def,
			Cache:      pricing.NewCache(cacheDir, cacheTTL, runtimeFetcher),
		}
	}

	return NewEstimatorWithRuntimes(registry, runtimes)
}

// NewEstimatorWithRuntimes creates an Estimator with explicit runtimes.
func NewEstimatorWithRuntimes(registry *handler.Registry, runtimes map[string]*ProviderRuntime) *Estimator {
	e := &Estimator{
		runtimes: runtimes,
	}
	e.resolver = NewCostResolver(registry, e)
	return e
}

// NewEstimatorWithResolver creates an Estimator with an explicit resolver and runtime map.
// Use this for custom registries, middleware, or testing.
func NewEstimatorWithResolver(runtimes map[string]*ProviderRuntime, resolver *CostResolver) *Estimator {
	return &Estimator{
		resolver: resolver,
		runtimes: runtimes,
	}
}

// CacheDir returns the resolved pricing cache directory path.
func (e *Estimator) CacheDir() string {
	for _, runtime := range e.runtimes {
		return runtime.Cache.Dir()
	}
	return ""
}

// SetPricingFetcher replaces the pricing fetcher (for testing or alternative providers).
func (e *Estimator) SetPricingFetcher(f pricing.PriceFetcher) {
	if len(e.runtimes) == 1 {
		for _, runtime := range e.runtimes {
			runtime.Cache.SetFetcher(f)
			return
		}
	}
	if runtime, ok := e.runtimes[awskit.ProviderID]; ok {
		runtime.Cache.SetFetcher(f)
	}
}

// CacheOldestAge returns the age of the oldest cache entry, or 0 if empty.
func (e *Estimator) CacheOldestAge() time.Duration {
	oldest := time.Duration(0)
	for _, runtime := range e.runtimes {
		age := runtime.Cache.OldestAge()
		if oldest == 0 || (age != 0 && age > oldest) {
			oldest = age
		}
	}
	return oldest
}

// CacheTTL returns the cache TTL.
func (e *Estimator) CacheTTL() time.Duration {
	for _, runtime := range e.runtimes {
		return runtime.Cache.TTL()
	}
	return 0
}

// CleanExpiredCache removes expired cache entries.
func (e *Estimator) CleanExpiredCache() {
	for providerID, runtime := range e.runtimes {
		if err := runtime.Cache.CleanExpired(); err != nil {
			log.WithError(err).WithField("provider", providerID).Debug("failed to clean expired cache")
		}
	}
}

// CacheEntries returns info about all cached pricing files.
func (e *Estimator) CacheEntries() []pricing.CacheEntry {
	var entries []pricing.CacheEntry
	for _, runtime := range e.runtimes {
		entries = append(entries, runtime.Cache.Entries()...)
	}
	return entries
}

// Resolver returns the underlying CostResolver for middleware registration.
func (e *Estimator) Resolver() *CostResolver { return e.resolver }

// GetIndex resolves pricing through the provider runtime selected by service id.
func (e *Estimator) GetIndex(ctx context.Context, service pricing.ServiceID, region string) (*pricing.PriceIndex, error) {
	runtime, ok := e.runtimes[service.Provider]
	if !ok {
		return nil, fmt.Errorf("no pricing runtime for provider %q", service.Provider)
	}
	return runtime.Cache.GetIndex(ctx, service, region)
}

// SourceName returns the configured price source for a provider.
func (e *Estimator) SourceName(providerID string) string {
	runtime, ok := e.runtimes[providerID]
	if !ok {
		return ""
	}
	return runtime.Definition.Manifest.PriceSource
}

// EstimateModule calculates cost for a single module from plan.json.
func (e *Estimator) EstimateModule(ctx context.Context, modulePath, region string) (*ModuleCost, error) {
	planJSONPath := filepath.Join(modulePath, "plan.json")

	parsedPlan, err := plan.ParseJSON(planJSONPath)
	if err != nil {
		return nil, fmt.Errorf("parse plan.json: %w", err)
	}

	result := &ModuleCost{
		ModuleID:   strings.ReplaceAll(modulePath, string(filepath.Separator), "/"),
		ModulePath: modulePath,
		Region:     region,
		Resources:  make([]ResourceCost, 0),
	}

	// Calculate costs for each resource
	for _, rc := range parsedPlan.Resources {
		attrs := rc.AfterValues
		if attrs == nil {
			attrs = rc.BeforeValues // delete has no after
		}

		req := ResolveRequest{
			ResourceType: handler.ResourceType(rc.Type),
			Address:      rc.Address,
			Name:         rc.Name,
			ModuleAddr:   rc.ModuleAddr,
			Region:       region,
			Attrs:        attrs,
		}

		// Resolve resource cost including any compound sub-resources
		costs := e.resolver.ResolveWithSubResources(ctx, req)

		for i := range costs {
			// For the primary resource on update/replace, calculate before-state cost
			if i == 0 && (rc.Action == plan.ActionUpdate || rc.Action == plan.ActionReplace) && rc.BeforeValues != nil {
				e.resolver.ResolveBeforeCost(ctx, &costs[i], handler.ResourceType(rc.Type), rc.BeforeValues, region)
			}

			result.Resources = append(result.Resources, costs[i])
			aggregateCost(result, costs[i], rc.Action)
		}
	}

	result.DiffCost = result.AfterCost - result.BeforeCost
	result.HasChanges = parsedPlan.HasChanges()
	result.Submodules = groupByModule(result.Resources)
	result.Provider, result.Providers = summarizeProviders(result.Resources)

	return result, nil
}

// aggregateCost adds a resource's cost to the module totals based on action.
func aggregateCost(result *ModuleCost, rc ResourceCost, action string) {
	if rc.IsUnsupported() {
		result.Unsupported++
		return
	}

	switch action {
	case plan.ActionCreate:
		result.AfterCost += rc.MonthlyCost
	case plan.ActionDelete:
		result.BeforeCost += rc.MonthlyCost
	case plan.ActionUpdate, plan.ActionReplace:
		result.BeforeCost += rc.BeforeMonthlyCost
		result.AfterCost += rc.MonthlyCost
	case plan.ActionNoOp:
		result.BeforeCost += rc.MonthlyCost
		result.AfterCost += rc.MonthlyCost
	}
}

// EstimateModules calculates costs for multiple modules concurrently.
func (e *Estimator) EstimateModules(ctx context.Context, modulePaths []string, regions map[string]string) (*EstimateResult, error) {
	const maxConcurrency = 4
	results := make([]ModuleCost, len(modulePaths))

	if prefetchErr := e.ValidateAndPrefetch(ctx, modulePaths, regions); prefetchErr != nil {
		log.WithError(prefetchErr).Warn("failed to prefetch some pricing data")
	}

	var g errgroup.Group
	g.SetLimit(maxConcurrency)

	for i, modulePath := range modulePaths {
		region := regions[modulePath]
		if region == "" {
			region = DefaultRegion
		}

		g.Go(func() error {
			mc, err := e.EstimateModule(ctx, modulePath, region)
			if err != nil {
				log.WithError(err).
					WithField("module", modulePath).
					Warn("failed to estimate module cost")
				results[i] = ModuleCost{
					ModuleID:   modulePath,
					ModulePath: modulePath,
					Error:      err.Error(),
				}
				return nil
			}
			results[i] = *mc
			return nil
		})
	}

	_ = g.Wait() //nolint:errcheck // individual errors collected in results

	result := &EstimateResult{
		Modules:          results,
		Currency:         "USD",
		GeneratedAt:      time.Now().UTC(),
		ProviderMetadata: e.providerMetadata(),
	}
	providerSet := make(map[string]bool)
	for i := range results {
		result.TotalBefore += results[i].BeforeCost
		result.TotalAfter += results[i].AfterCost
		for _, providerID := range results[i].Providers {
			providerSet[providerID] = true
		}
		if results[i].Error != "" {
			result.Errors = append(result.Errors, ModuleError{
				ModuleID: results[i].ModuleID,
				Error:    results[i].Error,
			})
		}
	}
	result.TotalDiff = result.TotalAfter - result.TotalBefore
	for providerID := range providerSet {
		result.Providers = append(result.Providers, providerID)
	}

	return result, nil
}

// ValidateAndPrefetch checks which pricing data is needed and downloads missing data.
//
//nolint:gocyclo // Aggregates plan parsing, handler resolution, and service prefetch planning in one orchestration path.
func (e *Estimator) ValidateAndPrefetch(ctx context.Context, modulePaths []string, regions map[string]string) error {
	requiredServices := make(map[pricing.ServiceID]map[string]bool)

	for _, modulePath := range modulePaths {
		planJSONPath := filepath.Join(modulePath, "plan.json")
		parsedPlan, err := plan.ParseJSON(planJSONPath)
		if err != nil {
			continue
		}

		region := regions[modulePath]
		if region == "" {
			region = DefaultRegion
		}

		for _, rc := range parsedPlan.Resources {
			resolved, ok := e.resolver.registry.Resolve(handler.ResourceType(rc.Type))
			if !ok || resolved.Handler.Category() != handler.CostCategoryStandard {
				continue
			}

			attrs := rc.AfterValues
			if attrs == nil {
				attrs = rc.BeforeValues
			}

			lookupBuilder, ok := resolved.Handler.(handler.LookupBuilder)
			if !ok {
				continue
			}

			lookup, err := lookupBuilder.BuildLookup(region, attrs)
			if err != nil || lookup == nil {
				continue
			}

			svc := lookup.ServiceID
			if requiredServices[svc] == nil {
				requiredServices[svc] = make(map[string]bool)
			}
			requiredServices[svc][region] = true
		}
	}

	services := make(map[pricing.ServiceID][]string)
	for svc, regionMap := range requiredServices {
		for region := range regionMap {
			services[svc] = append(services[svc], region)
		}
	}

	if len(services) == 0 {
		log.Warn("no supported resources found in plans — nothing to price")
		return nil
	}

	log.WithField("services", len(services)).Debug("required pricing services")

	var totalMissing int
	for providerID, runtime := range e.runtimes {
		providerServices := filterServicesForProvider(services, providerID)
		if len(providerServices) == 0 {
			continue
		}
		missing := runtime.Cache.Validate(providerServices)
		if len(missing) == 0 {
			continue
		}
		for i, m := range missing {
			totalMissing++
			log.WithField("provider", providerID).
				WithField("service", m.Service.String()).
				WithField("region", m.Region).
				WithField("progress", fmt.Sprintf("%d/%d", i+1, len(missing))).
				Info("downloading pricing data")
			if _, err := runtime.Cache.GetIndex(ctx, m.Service, m.Region); err != nil {
				return fmt.Errorf("fetch %s/%s pricing: %w", m.Service.String(), m.Region, err)
			}
		}
	}

	if totalMissing == 0 {
		log.Info("all pricing data is cached and valid")
	}

	return nil
}

func (e *Estimator) providerMetadata() map[string]ProviderMetadata {
	if len(e.runtimes) == 0 {
		return nil
	}

	meta := make(map[string]ProviderMetadata, len(e.runtimes))
	for providerID, runtime := range e.runtimes {
		if runtime.Definition.Manifest.ID == "" {
			continue
		}
		meta[providerID] = ProviderMetadata{
			DisplayName: runtime.Definition.Manifest.DisplayName,
			PriceSource: runtime.Definition.Manifest.PriceSource,
		}
	}
	return meta
}

func filterServicesForProvider(services map[pricing.ServiceID][]string, providerID string) map[pricing.ServiceID][]string {
	filtered := make(map[pricing.ServiceID][]string)
	for serviceID, regions := range services {
		if serviceID.Provider == providerID {
			filtered[serviceID] = regions
		}
	}
	return filtered
}

func summarizeProviders(resources []ResourceCost) (primary string, providers []string) {
	providerSet := make(map[string]bool)
	for i := range resources {
		resource := &resources[i]
		if resource.Provider != "" {
			providerSet[resource.Provider] = true
		}
	}

	if len(providerSet) == 0 {
		return "", nil
	}

	providers = make([]string, 0, len(providerSet))
	for providerID := range providerSet {
		providers = append(providers, providerID)
	}
	if len(providers) == 1 {
		return providers[0], providers
	}
	return "", providers
}
