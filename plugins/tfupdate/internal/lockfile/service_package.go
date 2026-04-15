package lockfile

import (
	"context"
	"errors"
	"os"

	"github.com/edelwud/terraci/plugins/tfupdate/internal/registrymeta"
)

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
