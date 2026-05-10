package diskblob

import (
	"context"
	"fmt"

	"github.com/edelwud/terraci/pkg/cache/blobcache"
	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/plugins/diskblob/internal/fsstore"
)

// Store is a filesystem-backed blob store.
type Store = fsstore.Store

// NewStore constructs a filesystem-backed blob store rooted at rootDir.
func NewStore(rootDir string) *Store {
	return fsstore.New(rootDir)
}

// NewBlobStore returns a new filesystem-backed blob store. Pass
// plugin.BlobStoreOptions{} to use defaults from configuration.
func (p *Plugin) NewBlobStore(_ context.Context, appCtx *plugin.AppContext, opts plugin.BlobStoreOptions) (blobcache.Store, error) {
	rootDir := resolveRootDir(appCtx, p.Config(), opts)
	if err := fsstore.ValidateRootDir(rootDir); err != nil {
		return nil, fmt.Errorf("diskblob: invalid root_dir: %w", err)
	}
	return NewStore(rootDir), nil
}
