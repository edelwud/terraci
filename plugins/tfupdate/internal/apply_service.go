package tfupdateengine

import (
	"context"

	"github.com/edelwud/terraci/plugins/tfupdate/internal/lockfile"
	tfregistry "github.com/edelwud/terraci/plugins/tfupdate/internal/registry"
)

// ApplyService applies discovered dependency updates to Terraform files.
type ApplyService struct {
	ctx      context.Context
	mutation dependencyMutationApplier
	pinner   dependencyPinner
	lockSync lockSyncApplier
	pin      bool
}

type applyServiceConfig struct {
	ctx           context.Context
	lockSyncer    lockfile.Syncer
	registry      tfregistry.Client
	downloader    lockfile.Downloader
	pin           bool
	lockPlatforms []string
}

// ApplyServiceOption configures ApplyService behavior.
type ApplyServiceOption interface {
	apply(*applyServiceConfig)
}

type applyServiceOptionFunc func(*applyServiceConfig)

func (f applyServiceOptionFunc) apply(cfg *applyServiceConfig) {
	f(cfg)
}

// NewApplyService constructs a stateless update apply service.
func NewApplyService(options ...ApplyServiceOption) *ApplyService {
	cfg := applyServiceConfig{ctx: context.Background()}
	for _, option := range options {
		if option != nil {
			option.apply(&cfg)
		}
	}
	if cfg.lockSyncer == nil {
		cfg.lockSyncer = lockfile.NewService(newProviderMetadataSource(cfg.registry), cfg.downloader, nil, cfg.lockPlatforms)
	}
	return &ApplyService{
		ctx:      cfg.ctx,
		mutation: dependencyMutationApplier{pin: cfg.pin},
		pinner:   dependencyPinner{},
		lockSync: lockSyncApplier{ctx: cfg.ctx, syncer: cfg.lockSyncer},
		pin:      cfg.pin,
	}
}

// WithApplyContext binds a cancellation context to follow-up lock updates.
func WithApplyContext(ctx context.Context) ApplyServiceOption {
	return applyServiceOptionFunc(func(cfg *applyServiceConfig) {
		if ctx != nil {
			cfg.ctx = ctx
		}
	})
}

// WithRegistryClient enables provider lock synchronization through registry metadata.
func WithRegistryClient(registry tfregistry.Client) ApplyServiceOption {
	return applyServiceOptionFunc(func(cfg *applyServiceConfig) {
		cfg.registry = registry
	})
}

// WithPackageDownloader overrides the provider package downloader for tests.
func WithPackageDownloader(downloader lockfile.Downloader) ApplyServiceOption {
	return applyServiceOptionFunc(func(cfg *applyServiceConfig) {
		cfg.downloader = downloader
	})
}

// WithProviderLockSyncer overrides provider lock synchronization behavior.
func WithProviderLockSyncer(syncer lockfile.Syncer) ApplyServiceOption {
	return applyServiceOptionFunc(func(cfg *applyServiceConfig) {
		cfg.lockSyncer = syncer
	})
}

// WithPinDependencies pins applied constraints to the selected exact version.
func WithPinDependencies(pin bool) ApplyServiceOption {
	return applyServiceOptionFunc(func(cfg *applyServiceConfig) {
		cfg.pin = pin
	})
}

// WithLockPlatforms restricts which platforms are hashed when syncing lock files.
// An empty list means all platforms.
func WithLockPlatforms(platforms []string) ApplyServiceOption {
	return applyServiceOptionFunc(func(cfg *applyServiceConfig) {
		cfg.lockPlatforms = platforms
	})
}

// Apply mutates the result items in place with apply outcomes.
func (s *ApplyService) Apply(result *UpdateResult) {
	for i := range result.Modules {
		if s.ctx != nil && s.ctx.Err() != nil {
			s.markRemainingApplyCanceled(result, i, 0)
			return
		}
		s.mutation.ApplyModule(&result.Modules[i])
	}

	for i := range result.Providers {
		if s.ctx != nil && s.ctx.Err() != nil {
			s.markRemainingApplyCanceled(result, len(result.Modules), i)
			return
		}
		s.mutation.ApplyProvider(&result.Providers[i])
	}

	if s.pin {
		s.pinner.PinModules(result)
		s.pinner.PinProviders(result)
	}

	s.lockSync.Apply(result, result.LockSync)
}

func (s *ApplyService) markRemainingApplyCanceled(result *UpdateResult, moduleIdx, providerIdx int) {
	issue := "apply canceled: " + s.ctx.Err().Error()

	for i := moduleIdx; i < len(result.Modules); i++ {
		if result.Modules[i].IsApplyPending() {
			result.Modules[i] = result.Modules[i].MarkError(issue)
		}
	}

	for i := providerIdx; i < len(result.Providers); i++ {
		if result.Providers[i].IsApplyPending() {
			result.Providers[i] = result.Providers[i].MarkError(issue)
		}
	}
}
