package lockfile

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"sync"

	log "github.com/caarlos0/log"
)

// Service materializes provider lock entries from registry metadata.
type Service struct {
	registry   ProviderMetadataSource
	downloader Downloader
	writer     Writer

	hashConcurrency int
	platforms       []string
	hashCache       map[string]string
	hashCacheMu     sync.Mutex
}

type providerLockTarget struct {
	lockPath    string
	address     ProviderAddress
	version     string
	constraints string
	platforms   []string
	h1Platforms []string
}

// NewService constructs a provider lock sync service.
// platforms restricts which platforms to hash; empty means all.
func NewService(registry ProviderMetadataSource, downloader Downloader, writer Writer, platforms []string) *Service {
	if downloader == nil {
		downloader = NewHTTPDownloader()
	}
	if writer == nil {
		writer = NewWriter()
	}

	return &Service{
		registry:        registry,
		downloader:      downloader,
		writer:          writer,
		hashConcurrency: hashConcurrencyLimit(),
		platforms:       platforms,
		hashCache:       make(map[string]string),
	}
}

// SyncProvider updates the lock file next to the given Terraform file.
func (s *Service) SyncProvider(ctx context.Context, req ProviderLockRequest) error {
	if s == nil || s.registry == nil {
		return nil
	}

	target, err := s.resolveProviderLockTarget(ctx, req)
	if err != nil {
		return err
	}

	log.WithField("provider", target.address.LockSource()).
		WithField("version", target.version).
		WithField("platforms", len(target.platforms)).
		WithField("download", len(target.h1Platforms)).
		Info("syncing provider lock file")

	newHashes, err := s.collectAllHashes(ctx, target.address, target.version, target.platforms, target.h1Platforms)
	if err != nil {
		return err
	}

	return s.writeProviderEntry(target.lockPath, LockProviderEntry{
		Source:      target.address.LockSource(),
		Version:     target.version,
		Constraints: target.constraints,
		Hashes:      LockHashSet(newHashes),
	})
}

func (s *Service) resolveProviderLockTarget(ctx context.Context, req ProviderLockRequest) (providerLockTarget, error) {
	lockPath := filepath.Join(filepath.Dir(filepath.Clean(req.TerraformFile)), ".terraform.lock.hcl")
	address, err := s.resolveProviderAddress(req.ProviderSource, lockPath)
	if err != nil {
		return providerLockTarget{}, fmt.Errorf("parse provider source %q: %w", req.ProviderSource, err)
	}

	allPlatforms, err := s.registry.ProviderPlatforms(ctx, address, req.Version)
	if err != nil {
		return providerLockTarget{}, fmt.Errorf("resolve provider platforms for %s %s: %w", req.ProviderSource, req.Version, err)
	}
	allPlatforms = normalizePlatforms(allPlatforms)
	if len(allPlatforms) == 0 {
		return providerLockTarget{}, fmt.Errorf("resolve provider platforms for %s %s: registry returned no platforms", req.ProviderSource, req.Version)
	}

	h1Platforms := allPlatforms
	if len(s.platforms) > 0 {
		h1Platforms = filterPlatforms(allPlatforms, s.platforms)
	}

	return providerLockTarget{
		lockPath:    lockPath,
		address:     address,
		version:     req.Version,
		constraints: req.Constraint,
		platforms:   allPlatforms,
		h1Platforms: h1Platforms,
	}, nil
}

func hashConcurrencyLimit() int {
	const maxConcurrency = 4

	if cpu := runtime.GOMAXPROCS(0); cpu > 0 && cpu < maxConcurrency {
		return cpu
	}

	return maxConcurrency
}
