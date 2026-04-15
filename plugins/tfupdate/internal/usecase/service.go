package usecase

import (
	"context"
	"fmt"
	"runtime"

	"github.com/caarlos0/log"
	"golang.org/x/sync/errgroup"

	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/parser"
	tfupdateengine "github.com/edelwud/terraci/plugins/tfupdate/internal"
	"github.com/edelwud/terraci/plugins/tfupdate/internal/domain"
	"github.com/edelwud/terraci/plugins/tfupdate/internal/lockfile"
	"github.com/edelwud/terraci/plugins/tfupdate/internal/planner"
	tfregistry "github.com/edelwud/terraci/plugins/tfupdate/internal/registry"
)

type Service struct {
	config     *tfupdateengine.UpdateConfig
	parser     *parser.Parser
	registry   tfregistry.Client
	downloader lockfile.Downloader
	write      bool
}

func New(
	config *tfupdateengine.UpdateConfig,
	moduleParser *parser.Parser,
	registry tfregistry.Client,
	downloader lockfile.Downloader,
	write bool,
) *Service {
	return &Service{
		config:     config,
		parser:     moduleParser,
		registry:   registry,
		downloader: downloader,
		write:      write,
	}
}

func (s *Service) Run(ctx context.Context, modules []*discovery.Module) (*tfupdateengine.UpdateResult, error) {
	builder := tfupdateengine.NewUpdateResultBuilder()
	results, err := s.planModules(ctx, modules, builder)
	if err != nil {
		return nil, fmt.Errorf("walk modules: %w", err)
	}

	for i := range results {
		result := &results[i]
		if result.parseError {
			continue
		}
		for j := range result.result.Modules {
			builder.AddModuleUpdate(result.result.Modules[j])
		}
		for j := range result.result.Providers {
			builder.AddProviderUpdate(result.result.Providers[j])
		}
		for _, plan := range result.result.LockSync {
			builder.AddLockSyncPlan(plan)
		}
	}

	final := builder.Result()
	if s.write {
		tfupdateengine.NewApplyService(
			tfupdateengine.WithApplyContext(ctx),
			tfupdateengine.WithRegistryClient(s.registry),
			tfupdateengine.WithPackageDownloader(s.downloader),
			tfupdateengine.WithPinDependencies(s.config.PinEnabled()),
			tfupdateengine.WithLockPlatforms(s.config.LockPlatforms()),
		).Apply(final)
	}

	final.Summary = tfupdateengine.BuildUpdateSummary(final)
	return final, nil
}

type moduleRunResult struct {
	result     *tfupdateengine.UpdateResult
	parseError bool
}

func (s *Service) planModules(
	ctx context.Context,
	modules []*discovery.Module,
	builder *tfupdateengine.UpdateResultBuilder,
) ([]moduleRunResult, error) {
	results := make([]moduleRunResult, len(modules))
	var group errgroup.Group
	group.SetLimit(checkConcurrencyLimit())

	for i := range modules {
		group.Go(func() error {
			if err := ctx.Err(); err != nil {
				return err
			}
			parsed, err := s.parser.ParseModule(ctx, modules[i].Path)
			if err != nil {
				log.WithField("module", modules[i].RelativePath).WithError(err).Warn("failed to parse module")
				builder.RecordError()
				results[i] = moduleRunResult{parseError: true}
				return nil
			}

			plan, err := planner.New(ctx, s.config, s.registry).SolveModule(modules[i], parsed)
			if err != nil {
				log.WithField("module", modules[i].RelativePath).WithError(err).Warn("failed to solve module plan")
				builder.RecordError()
				results[i] = moduleRunResult{parseError: true}
				return nil
			}

			results[i] = moduleRunResult{
				result: mapPlanToResult(plan),
			}
			return nil
		})
	}

	if err := group.Wait(); err != nil {
		return nil, err
	}
	return results, nil
}

func mapPlanToResult(plan *domain.ModulePlan) *tfupdateengine.UpdateResult {
	result := tfupdateengine.NewUpdateResult()
	if plan == nil {
		return result
	}

	for i := range plan.Modules {
		module := &plan.Modules[i]
		update := domain.NewModuleVersionUpdate(module.Dependency)
		update.File = module.File
		update.CurrentVersion = module.Current
		update.LatestVersion = module.Latest
		update.BumpedVersion = module.Selected
		update.Status = module.Status
		update.Issue = module.Issue
		update.ProviderDeps = module.ProviderDeps
		result.Modules = append(result.Modules, update)
	}

	for i := range plan.Providers {
		provider := &plan.Providers[i]
		update := domain.NewProviderVersionUpdate(provider.Dependency)
		update.File = provider.File
		update.CurrentVersion = provider.Current
		update.LatestVersion = provider.Latest
		update.BumpedVersion = provider.Selected
		update.Status = provider.Status
		update.Issue = provider.Issue
		result.Providers = append(result.Providers, update)
	}
	if len(plan.LockSync.Providers) > 0 {
		result.LockSync = append(result.LockSync, plan.LockSync)
	}

	return result
}

func checkConcurrencyLimit() int {
	const maxConcurrency = 8
	if cpu := runtime.GOMAXPROCS(0); cpu > 0 && cpu < maxConcurrency {
		return cpu
	}
	return maxConcurrency
}
