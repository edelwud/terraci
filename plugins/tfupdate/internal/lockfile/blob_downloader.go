package lockfile

import (
	"context"
	"fmt"
	"os"

	"github.com/edelwud/terraci/pkg/plugin"
)

type blobCachingDownloader struct {
	base      Downloader
	store     plugin.BlobStore
	namespace string
}

// NewBlobCachingDownloader wraps a downloader with blob-store backed artifact caching.
func NewBlobCachingDownloader(base Downloader, store plugin.BlobStore, namespace string) Downloader {
	if base == nil {
		base = NewHTTPDownloader()
	}
	if store == nil {
		return base
	}
	return &blobCachingDownloader{
		base:      base,
		store:     store,
		namespace: namespace,
	}
}

func (d *blobCachingDownloader) Download(ctx context.Context, url, destPath string) error {
	return d.base.Download(ctx, url, destPath)
}

func (d *blobCachingDownloader) DownloadCached(ctx context.Context, cacheKey, url, destPath string) error {
	if payload, ok, _, err := d.store.Get(ctx, d.namespace, cacheKey); err == nil && ok {
		return os.WriteFile(destPath, payload, 0o600)
	}

	if err := d.base.Download(ctx, url, destPath); err != nil {
		return err
	}

	payload, err := os.ReadFile(destPath)
	if err != nil {
		return fmt.Errorf("read downloaded artifact %s: %w", destPath, err)
	}
	if _, err := d.store.Put(ctx, d.namespace, cacheKey, payload, plugin.PutBlobOptions{
		ContentType: "application/zip",
	}); err != nil {
		return fmt.Errorf("cache provider artifact %q: %w", cacheKey, err)
	}

	return nil
}
