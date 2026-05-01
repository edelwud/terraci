// Package diskblob provides a filesystem-backed blob store backend.
package diskblob

import (
	"context"
	"fmt"

	"github.com/edelwud/terraci/pkg/cache/blobcache"
	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/pkg/plugin/registry"
	"github.com/edelwud/terraci/plugins/diskblob/internal/fsstore"
)

func init() {
	registry.RegisterFactory(func() plugin.Plugin {
		return &Plugin{BasePlugin: plugin.BasePlugin[*Config]{
			PluginName: "diskblob",
			PluginDesc: "Filesystem-backed blob store backend",
			EnableMode: plugin.EnabledByDefault,
			DefaultCfg: func() *Config { return &Config{Enabled: true} },
			IsEnabledFn: func(cfg *Config) bool {
				return cfg == nil || cfg.Enabled
			},
		}}
	})
}

// Plugin is the filesystem-backed blob store backend.
type Plugin struct {
	plugin.BasePlugin[*Config]
}

// NewBlobStore returns a new filesystem-backed blob store.
func (p *Plugin) NewBlobStore(_ context.Context, appCtx *plugin.AppContext) (blobcache.Store, error) {
	return p.NewBlobStoreWithOptions(context.Background(), appCtx, plugin.BlobStoreOptions{})
}

// NewBlobStoreWithOptions returns a new filesystem-backed blob store with optional overrides.
func (p *Plugin) NewBlobStoreWithOptions(_ context.Context, appCtx *plugin.AppContext, opts plugin.BlobStoreOptions) (blobcache.Store, error) {
	rootDir := resolveRootDir(appCtx, p.Config(), opts)
	if err := fsstore.ValidateRootDir(rootDir); err != nil {
		return nil, fmt.Errorf("diskblob: invalid root_dir: %w", err)
	}
	return NewStore(rootDir), nil
}
