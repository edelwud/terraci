package lockfile

import (
	"context"
	"fmt"
	"strings"

	"golang.org/x/sync/errgroup"

	"github.com/edelwud/terraci/pkg/log"
)

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

			pkg, err := s.registry.ProviderPackage(groupCtx, address, version, platform)
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
