package diskblob

import (
	"path/filepath"

	"github.com/edelwud/terraci/pkg/plugin"
)

// Config controls the filesystem-backed blob store backend.
type Config struct {
	Enabled bool   `yaml:"enabled,omitempty" json:"enabled,omitempty" jsonschema:"description=Enable the built-in filesystem blob store backend,default=true"`
	RootDir string `yaml:"root_dir,omitempty" json:"root_dir,omitempty" jsonschema:"description=Directory where blob objects are stored,default=~/.terraci/blobs"`
}

func resolveRootDir(appCtx *plugin.AppContext, cfg *Config, opts plugin.BlobStoreOptions) string {
	if opts.RootDir != "" {
		return opts.RootDir
	}
	if cfg != nil && cfg.RootDir != "" {
		return cfg.RootDir
	}
	if appCtx != nil && appCtx.ServiceDir() != "" {
		return filepath.Join(appCtx.ServiceDir(), "blobs")
	}
	return filepath.Join(userHomeDir(), ".terraci", "blobs")
}
