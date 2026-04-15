package lockfile

import (
	"context"

	"github.com/edelwud/terraci/plugins/tfupdate/internal/registrymeta"
)

// ProviderLockRequest describes the target provider lock state to materialize.
type ProviderLockRequest struct {
	ProviderSource string
	Version        string
	Constraint     string
	TerraformFile  string
}

// ProviderMetadataSource exposes provider package metadata needed to construct lock entries.
type ProviderMetadataSource interface {
	ProviderPlatforms(ctx context.Context, address ProviderAddress, version string) ([]string, error)
	ProviderPackage(ctx context.Context, address ProviderAddress, version, platform string) (*registrymeta.ProviderPackage, error)
}

// Downloader retrieves provider zip archives to a local path.
type Downloader interface {
	Download(ctx context.Context, url, destPath string) error
}

// CachingDownloader optionally accepts a stable cache key for artifact reuse.
type CachingDownloader interface {
	Downloader
	DownloadCached(ctx context.Context, cacheKey, url, destPath string) error
}

// Writer persists lock documents.
type Writer interface {
	WriteDocument(filePath string, doc *LockDocument) error
}

// Syncer updates provider lock files from registry metadata.
type Syncer interface {
	SyncProvider(ctx context.Context, req ProviderLockRequest) error
}

// LockDocument describes the typed state of .terraform.lock.hcl.
type LockDocument struct {
	Providers []LockProviderEntry
}

func (d *LockDocument) Provider(source string) *LockProviderEntry {
	if d == nil {
		return nil
	}
	for i := range d.Providers {
		if d.Providers[i].Source == source {
			return &d.Providers[i]
		}
	}
	return nil
}

func (d *LockDocument) UpsertProvider(entry LockProviderEntry) {
	if d == nil {
		return
	}
	for i := range d.Providers {
		if d.Providers[i].Source == entry.Source {
			d.Providers[i] = entry
			return
		}
	}
	d.Providers = append(d.Providers, entry)
}

// LockProviderEntry describes a provider block in .terraform.lock.hcl.
type LockProviderEntry struct {
	Source      string
	Version     string
	Constraints string
	Hashes      LockHashSet
}

// LockedProviderEntry is a read-only view used by apply/report flows.
type LockedProviderEntry = LockProviderEntry

func (e LockProviderEntry) H1Count() int {
	return e.Hashes.CountByPrefix("h1:")
}

// LockHashSet stores normalized provider hashes.
type LockHashSet []string

func (s LockHashSet) CountByPrefix(prefix string) int {
	var count int
	for _, item := range s {
		if len(item) >= len(prefix) && item[:len(prefix)] == prefix {
			count++
		}
	}
	return count
}

func (s LockHashSet) Merge(other []string) LockHashSet {
	return LockHashSet(normalizeHashes(append(append([]string(nil), s...), other...)))
}
