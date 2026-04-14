package tfupdateengine

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/caarlos0/log"

	"github.com/edelwud/terraci/plugins/tfupdate/internal/lockfile"
	"github.com/edelwud/terraci/plugins/tfupdate/internal/tfwrite"
)

// ApplyService applies discovered dependency updates to Terraform files.
type ApplyService struct {
	ctx        context.Context
	lockSyncer lockfile.Syncer
	registry   RegistryClient
	pin        bool
}

type applyServiceConfig struct {
	ctx           context.Context
	lockSyncer    lockfile.Syncer
	registry      RegistryClient
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
		cfg.lockSyncer = lockfile.NewService(cfg.registry, cfg.downloader, nil, cfg.lockPlatforms)
	}
	return &ApplyService{
		ctx:        cfg.ctx,
		lockSyncer: cfg.lockSyncer,
		registry:   cfg.registry,
		pin:        cfg.pin,
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
func WithRegistryClient(registry RegistryClient) ApplyServiceOption {
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
		s.applyModuleUpdate(&result.Modules[i])
	}

	for i := range result.Providers {
		if s.ctx != nil && s.ctx.Err() != nil {
			s.markRemainingApplyCanceled(result, len(result.Modules), i)
			return
		}
		s.applyProviderUpdate(&result.Providers[i])
	}

	if s.pin {
		s.pinUpToDateModules(result)
		s.pinUpToDateProviders(result)
	}

	s.syncLockPlans(result, result.LockSync)
}

func (s *ApplyService) applyModuleUpdate(update *ModuleVersionUpdate) {
	if !update.IsApplyPending() {
		return
	}

	newConstraint, ok := buildAppliedConstraint(update.BumpedVersion, update.Constraint(), s.pin)
	if !ok {
		s.markModuleApplyError(update, "failed to build bumped module constraint", nil)
		return
	}

	if err := tfwrite.WriteModuleVersion(update.File, update.CallName(), newConstraint); err != nil {
		s.markModuleApplyError(update, fmt.Sprintf("write module version: %v", err), err)
		return
	}

	*update = update.MarkApplied()
}

func (s *ApplyService) applyProviderUpdate(update *ProviderVersionUpdate) {
	if !update.IsApplyPending() {
		return
	}
	if strings.TrimSpace(update.File) == "" {
		return
	}

	newConstraint, ok := buildAppliedConstraint(update.BumpedVersion, update.Constraint(), s.pin)
	if !ok {
		s.markProviderApplyError(update, "failed to build bumped provider constraint", nil)
		return
	}

	if err := tfwrite.WriteProviderVersion(update.File, update.ProviderName(), newConstraint); err != nil {
		s.markProviderApplyError(update, fmt.Sprintf("write provider version: %v", err), err)
		return
	}

	*update = update.MarkApplied()
}

func parseVersionOrZero(s string) Version {
	v, err := ParseVersion(s)
	if err != nil {
		return Version{}
	}
	return v
}

func buildAppliedConstraint(bumpedVersion, originalConstraint string, pin bool) (string, bool) {
	version := parseVersionOrZero(bumpedVersion)
	if version.IsZero() {
		return "", false
	}
	if pin {
		return version.String(), true
	}
	return BumpConstraint(originalConstraint, version), true
}

func (s *ApplyService) markModuleApplyError(update *ModuleVersionUpdate, issue string, err error) {
	*update = update.MarkError(issue)
	entry := log.WithField("module", update.ModulePath())
	if err != nil {
		entry = entry.WithError(err)
	}
	entry.Warn(issue)
}

func (s *ApplyService) markProviderApplyError(update *ProviderVersionUpdate, issue string, err error) {
	*update = update.MarkError(issue)
	entry := log.WithField("provider", update.ProviderSource())
	if err != nil {
		entry = entry.WithError(err)
	}
	entry.Warn(issue)
}

func (s *ApplyService) pinUpToDateProviders(result *UpdateResult) {
	for i := range result.Providers {
		p := &result.Providers[i]
		if p.Status != StatusUpToDate {
			continue
		}
		if strings.TrimSpace(p.File) == "" || p.CurrentVersion == "" {
			continue
		}
		if isExactConstraint(p.Constraint(), p.CurrentVersion) {
			continue
		}

		if err := tfwrite.WriteProviderVersion(p.File, p.ProviderName(), p.CurrentVersion); err != nil {
			log.WithField("provider", p.ProviderSource()).WithError(err).Warn("failed to pin provider version")
			continue
		}

		log.WithField("provider", p.ProviderSource()).
			WithField("version", p.CurrentVersion).
			Info("pinned provider version")
	}
}

func (s *ApplyService) pinUpToDateModules(result *UpdateResult) {
	for i := range result.Modules {
		m := &result.Modules[i]
		if m.Status != StatusUpToDate {
			continue
		}
		if strings.TrimSpace(m.File) == "" {
			continue
		}

		targetVersion := m.BumpedVersion
		if targetVersion == "" {
			targetVersion = m.CurrentVersion
		}
		if targetVersion == "" || isExactConstraint(m.Constraint(), targetVersion) {
			continue
		}

		if err := tfwrite.WriteModuleVersion(m.File, m.CallName(), targetVersion); err != nil {
			log.WithField("module", m.CallName()).WithError(err).Warn("failed to pin module version")
			continue
		}

		log.WithField("module", m.CallName()).
			WithField("version", targetVersion).
			Info("pinned module version")
	}
}

func isExactConstraint(constraint, version string) bool {
	cs, err := ParseConstraints(constraint)
	if err != nil || len(cs) != 1 {
		return false
	}
	return cs[0].Op == OpEqual && cs[0].Version.String() == version
}

func (s *ApplyService) syncLockPlans(result *UpdateResult, plans []LockSyncPlan) {
	for _, plan := range plans {
		for _, provider := range plan.Providers {
			if s.ctx != nil && s.ctx.Err() != nil {
				return
			}
			if provider.ProviderSource == "" || provider.Version == "" || provider.TerraformFile == "" {
				continue
			}

			log.WithField("provider", provider.ProviderSource).
				WithField("version", provider.Version).
				Info("syncing provider lock file")

			if err := s.lockSyncer.SyncProvider(s.ctx, lockfile.ProviderLockRequest{
				ProviderSource: provider.ProviderSource,
				Version:        provider.Version,
				Constraint:     provider.Constraint,
				TerraformFile:  provider.TerraformFile,
			}); err != nil {
				s.markLockSyncError(result, provider, err)
				log.WithField("provider", provider.ProviderSource).WithError(err).Warn("failed to sync lock file")
				continue
			}
			s.markLockSyncApplied(result, provider)
		}
	}
}

func (s *ApplyService) markLockSyncError(result *UpdateResult, provider LockProviderSync, err error) {
	if result == nil {
		return
	}

	issue := fmt.Sprintf("update provider lock file: %v", err)
	for i := range result.Providers {
		update := &result.Providers[i]
		if update.ProviderSource() != provider.ProviderSource || update.File != provider.TerraformFile {
			continue
		}
		*update = update.MarkError(issue)
		return
	}
}

func (s *ApplyService) markLockSyncApplied(result *UpdateResult, provider LockProviderSync) {
	if result == nil {
		return
	}

	for i := range result.Providers {
		update := &result.Providers[i]
		if update.ProviderSource() != provider.ProviderSource {
			continue
		}
		if strings.TrimSpace(update.File) == "" || update.File == provider.TerraformFile {
			*update = update.MarkApplied()
			return
		}
	}
}

// mergeConstraints combines a root constraint with additional constraints from
// child modules, producing a comma-separated deduplicated normalized string
// matching Terraform's lock file format.
func MergeConstraints(root string, extras []string) string {
	all := make([]string, 0, 1+len(extras))
	seen := make(map[string]struct{})

	for _, c := range append(extras, root) {
		for _, normalized := range normalizeConstraintParts(c) {
			if _, ok := seen[normalized]; ok {
				continue
			}
			seen[normalized] = struct{}{}
			all = append(all, normalized)
		}
	}

	if len(all) == 0 {
		return root
	}

	sort.Slice(all, constraintLess(all))
	return strings.Join(all, ", ")
}

var constraintOpStr = map[ConstraintOp]string{
	OpEqual:        "",
	OpNotEqual:     "!= ",
	OpGreater:      "> ",
	OpGreaterEqual: ">= ",
	OpLess:         "< ",
	OpLessEqual:    "<= ",
	OpPessimistic:  "~> ",
}

// constraintLess returns a sort function that orders constraints by version
// (ascending), with operator constraints before exact versions at the same version.
func constraintLess(items []string) func(i, j int) bool {
	type parsed struct {
		v     Version
		hasOp bool
	}
	cache := make([]parsed, len(items))
	for idx, s := range items {
		c, err := parseSingleConstraint(s)
		if err != nil {
			continue
		}
		cache[idx] = parsed{v: c.Version, hasOp: c.Op != OpEqual}
	}

	return func(i, j int) bool {
		ci, cj := cache[i], cache[j]
		if cmp := ci.v.Compare(cj.v); cmp != 0 {
			return cmp < 0
		}
		// Same version: operator constraints before exact.
		if ci.hasOp != cj.hasOp {
			return ci.hasOp
		}
		return items[i] < items[j]
	}
}

// normalizeConstraintParts parses a (possibly comma-separated) constraint string
// and returns each part in normalized form: ">= 5.79" → ">= 5.79.0".
// Pessimistic constraints preserve their original precision: "~> 5.2" stays "~> 5.2".
func normalizeConstraintParts(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}

	constraints, err := ParseConstraints(s)
	if err != nil {
		return []string{s}
	}

	out := make([]string, 0, len(constraints))
	for _, c := range constraints {
		prefix := constraintOpStr[c.Op]
		ver := c.Version.String() // always "major.minor.patch"

		// For pessimistic (~>) with 2 parts, keep "major.minor" form.
		if c.Op == OpPessimistic && c.Parts <= 2 {
			ver = fmt.Sprintf("%d.%d", c.Version.Major, c.Version.Minor)
		}

		out = append(out, prefix+ver)
	}
	return out
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
