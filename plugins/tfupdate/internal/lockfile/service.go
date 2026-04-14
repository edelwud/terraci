package lockfile

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"golang.org/x/sync/errgroup"

	"github.com/edelwud/terraci/pkg/log"
	"github.com/edelwud/terraci/plugins/tfupdate/internal/registrymeta"
	"github.com/edelwud/terraci/plugins/tfupdate/internal/sourceaddr"
)

// Service materializes provider lock entries from registry metadata.
type Service struct {
	registry   Registry
	downloader Downloader
	writer     Writer

	hashConcurrency int
	platforms       []string
	hashCache       map[string]string
	hashCacheMu     sync.Mutex
}

// NewService constructs a provider lock sync service.
// platforms restricts which platforms to hash; empty means all.
func NewService(registry Registry, downloader Downloader, writer Writer, platforms []string) *Service {
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

	lockPath := filepath.Join(filepath.Dir(filepath.Clean(req.TerraformFile)), ".terraform.lock.hcl")
	address, err := s.resolveProviderAddress(req.ProviderSource, lockPath)
	if err != nil {
		return fmt.Errorf("parse provider source %q: %w", req.ProviderSource, err)
	}

	allPlatforms, err := s.registry.ProviderPlatforms(ctx, address.Hostname, address.Namespace, address.Type, req.Version)
	if err != nil {
		return fmt.Errorf("resolve provider platforms for %s %s: %w", req.ProviderSource, req.Version, err)
	}
	allPlatforms = normalizePlatforms(allPlatforms)
	if len(allPlatforms) == 0 {
		return fmt.Errorf("resolve provider platforms for %s %s: registry returned no platforms", req.ProviderSource, req.Version)
	}

	// h1 hashes require downloading ZIP files — only for selected platforms.
	// zh hashes use shasum from registry metadata — collected for all platforms.
	h1Platforms := allPlatforms
	if len(s.platforms) > 0 {
		h1Platforms = filterPlatforms(allPlatforms, s.platforms)
	}

	log.WithField("provider", address.LockSource()).
		WithField("version", req.Version).
		WithField("platforms", len(allPlatforms)).
		WithField("download", len(h1Platforms)).
		Info("syncing provider lock file")

	newHashes, err := s.collectAllHashes(ctx, address, req.Version, allPlatforms, h1Platforms)
	if err != nil {
		return err
	}

	doc, err := ParseDocument(lockPath)
	if err != nil {
		return err
	}

	entry := LockProviderEntry{
		Source:      address.LockSource(),
		Version:     req.Version,
		Constraints: req.Constraint,
		Hashes:      LockHashSet(newHashes),
	}
	if existing := doc.Provider(entry.Source); existing != nil {
		entry.Hashes = existing.Hashes.Merge(newHashes)
	}
	doc.UpsertProvider(entry)

	if err := s.writer.WriteDocument(lockPath, doc); err != nil {
		return fmt.Errorf("write provider lock file %s: %w", lockPath, err)
	}

	return nil
}

func (s *Service) collectAllHashes(
	ctx context.Context,
	address ProviderAddress,
	version string,
	allPlatforms []string,
	h1Platforms []string,
) ([]string, error) {
	h1Set := make(map[string]struct{}, len(h1Platforms))
	for _, p := range h1Platforms {
		h1Set[p] = struct{}{}
	}

	group, groupCtx := errgroup.WithContext(ctx)
	group.SetLimit(s.hashConcurrency)

	type platformResult struct {
		zh string
		h1 string
	}
	results := make([]platformResult, len(allPlatforms))

	for i, platform := range allPlatforms {
		_, needH1 := h1Set[platform]
		group.Go(func() error {
			if err := groupCtx.Err(); err != nil {
				return err
			}

			pkg, err := s.registry.ProviderPackage(groupCtx, address.Hostname, address.Namespace, address.Type, version, platform)
			if err != nil {
				return fmt.Errorf("resolve provider package for %s %s %s: %w", address.LockSource(), version, platform, err)
			}
			if pkg == nil {
				return fmt.Errorf("resolve provider package for %s %s %s: package metadata is nil", address.LockSource(), version, platform)
			}

			if pkg.Shasum != "" {
				results[i].zh = "zh:" + strings.ToLower(pkg.Shasum)
			}

			if needH1 {
				h1, err := s.cachedPackageHash(groupCtx, address, version, platform, pkg)
				if err != nil {
					return fmt.Errorf("hash provider package for %s %s %s: %w", address.LockSource(), version, platform, err)
				}
				results[i].h1 = h1

				log.WithField("provider", address.LockSource()).
					WithField("platform", platform).
					Debug("hashed platform package")
			}

			return nil
		})
	}

	if err := group.Wait(); err != nil {
		return nil, err
	}

	hashes := make([]string, 0, len(allPlatforms)+len(h1Platforms))
	for _, r := range results {
		if r.h1 != "" {
			hashes = append(hashes, r.h1)
		}
		if r.zh != "" {
			hashes = append(hashes, r.zh)
		}
	}

	return normalizeHashes(hashes), nil
}

func (s *Service) resolveProviderAddress(source, lockPath string) (ProviderAddress, error) {
	address, err := ParseProviderAddress(source)
	if err != nil {
		return ProviderAddress{}, err
	}

	namespace, typeName, shortErr := sourceaddr.ParseProviderSource(source)
	if shortErr != nil {
		return address, nil //nolint:nilerr // short source parse failure is non-fatal; fall back to default address
	}

	existing, status := lookupLockedProviderAddress(lockPath, namespace, typeName)
	switch status {
	case lockAddressFound:
		return existing, nil
	case lockAddressAmbiguous:
	case lockAddressNotFound:
		log.WithField("provider_source", source).
			WithField("lock_file", lockPath).
			Warn("update: multiple matching provider lock entries found; using default hostname resolution")
	}

	return address, nil
}

type lockAddressStatus int

const (
	lockAddressNotFound lockAddressStatus = iota
	lockAddressFound
	lockAddressAmbiguous
)

func lookupLockedProviderAddress(lockPath, namespace, typeName string) (ProviderAddress, lockAddressStatus) {
	doc, err := ParseDocument(lockPath)
	if err != nil {
		return ProviderAddress{}, lockAddressNotFound
	}

	var matched ProviderAddress
	var found bool
	for _, provider := range doc.Providers {
		address, err := ParseProviderAddress(provider.Source)
		if err != nil {
			continue
		}
		if address.Namespace != namespace || address.Type != typeName {
			continue
		}

		if found {
			return ProviderAddress{}, lockAddressAmbiguous
		}
		matched = address
		found = true
	}

	if found {
		return matched, lockAddressFound
	}

	return ProviderAddress{}, lockAddressNotFound
}

func (s *Service) downloadAndHashPackage(ctx context.Context, pkg *registrymeta.ProviderPackage, cacheKey string) (string, error) {
	if pkg == nil {
		return "", errors.New("provider package metadata is nil")
	}
	if pkg.DownloadURL == "" {
		return "", errors.New("provider package download URL is empty")
	}

	tmpFile, err := os.CreateTemp("", "terraci-provider-*.zip")
	if err != nil {
		return "", err
	}
	tmpPath := tmpFile.Name()
	if err := tmpFile.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return "", err
	}
	defer os.Remove(tmpPath)

	if downloader, ok := s.downloader.(CachingDownloader); ok {
		if err := downloader.DownloadCached(ctx, cacheKey, pkg.DownloadURL, tmpPath); err != nil {
			return "", err
		}
	} else if err := s.downloader.Download(ctx, pkg.DownloadURL, tmpPath); err != nil {
		return "", err
	}

	if pkg.Shasum != "" {
		if err := verifyPackageChecksum(tmpPath, pkg.Shasum); err != nil {
			return "", err
		}
	}

	return hashZip(tmpPath)
}

func (s *Service) cachedPackageHash(
	ctx context.Context,
	address ProviderAddress,
	version string,
	platform string,
	pkg *registrymeta.ProviderPackage,
) (string, error) {
	key := packageHashCacheKey(address, version, platform)

	s.hashCacheMu.Lock()
	hash, ok := s.hashCache[key]
	s.hashCacheMu.Unlock()
	if ok {
		return hash, nil
	}

	hash, err := s.downloadAndHashPackage(ctx, pkg, key+"/archive")
	if err != nil {
		return "", err
	}

	s.hashCacheMu.Lock()
	s.hashCache[key] = hash
	s.hashCacheMu.Unlock()
	return hash, nil
}

func packageHashCacheKey(address ProviderAddress, version, platform string) string {
	return address.LockSource() + "@" + version + "/" + platform
}

func hashConcurrencyLimit() int {
	const maxConcurrency = 4

	if cpu := runtime.GOMAXPROCS(0); cpu > 0 && cpu < maxConcurrency {
		return cpu
	}

	return maxConcurrency
}
