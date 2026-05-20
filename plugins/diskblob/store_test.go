package diskblob

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/edelwud/terraci/pkg/cache/blobcache"
	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/pkg/plugin/plugintest"
)

func TestStore_ContractSuite(t *testing.T) {
	plugintest.RunBlobStoreContractSuite(t, plugintest.BlobStoreContractSuite{
		Factory: func(tb testing.TB, backingID string) blobcache.Store {
			tb.Helper()
			return NewStore(backingID)
		},
		Namespace:    "cost/pricing",
		AltNamespace: "cost/alt",
		Key:          "aws/AmazonEC2/us-east-1.json",
		AltKey:       "aws/AmazonRDS/us-east-1.json",
		Payload:      []byte("hello"),
		StreamData:   "streamed",
	})
}

func TestPlugin_BaseConfigContract(t *testing.T) {
	p := &Plugin{
		BasePlugin: plugin.BasePlugin[*Config]{
			PluginName: "diskblob",
			PluginDesc: "Filesystem-backed blob/object cache backend",
			EnableMode: plugin.EnabledByDefault,
			DefaultCfg: func() *Config { return &Config{Enabled: true} },
			IsEnabledFn: func(cfg *Config) bool {
				return cfg == nil || cfg.Enabled
			},
		},
	}

	plugintest.AssertBaseConfigPlugin[*Config](t, plugintest.BaseConfigPluginContract[*Config]{
		Plugin:     p,
		Default:    &Config{Enabled: true},
		Configured: &Config{Enabled: true, RootDir: "/tmp/blob-cache"},
		Decoded:    &Config{Enabled: false, RootDir: "/tmp/decoded"},
		Mutate: func(c *Config) {
			if c != nil {
				c.Enabled = !c.Enabled
				c.RootDir = "mutated"
			}
		},
		Equal: func(got, want *Config) bool {
			if got == nil || want == nil {
				return got == want
			}
			return *got == *want
		},
	})
}

func TestPlugin_BlobStoreProviderContract(t *testing.T) {
	p := &Plugin{
		BasePlugin: plugin.BasePlugin[*Config]{
			PluginName: "diskblob",
			PluginDesc: "Filesystem-backed blob/object cache backend",
			EnableMode: plugin.EnabledByDefault,
			DefaultCfg: func() *Config { return &Config{Enabled: true} },
			IsEnabledFn: func(cfg *Config) bool {
				return cfg == nil || cfg.Enabled
			},
		},
	}

	plugintest.AssertBlobStoreProvider(t, plugintest.BlobStoreProviderContract{
		Provider: p,
		Options:  plugin.BlobStoreOptions{RootDir: t.TempDir()},
	})
}

func TestStore_DescribeBlobStore(t *testing.T) {
	store := NewStore(t.TempDir())
	info := store.DescribeBlobStore()
	if info.Backend != "diskblob" {
		t.Fatalf("DescribeBlobStore().Backend = %q, want diskblob", info.Backend)
	}
	if info.Root != store.BlobStoreRootDir() {
		t.Fatalf("DescribeBlobStore().Root = %q, want %q", info.Root, store.BlobStoreRootDir())
	}
	if !info.SupportsList || !info.SupportsStream || !info.SupportsDeleteNamespace {
		t.Fatalf("DescribeBlobStore() capabilities = %+v, want all supported", info)
	}
}

func TestStore_CheckBlobStore_Succeeds(t *testing.T) {
	store := NewStore(t.TempDir())
	if err := store.CheckBlobStore(context.Background()); err != nil {
		t.Fatalf("CheckBlobStore() error = %v", err)
	}
}

func TestStore_CheckBlobStore_UnusableRoot(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "blob-root")
	if err := os.WriteFile(filePath, []byte("not a dir"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	store := NewStore(filePath)
	err := store.CheckBlobStore(context.Background())
	if err == nil || !strings.Contains(err.Error(), "diskblob: health check failed") {
		t.Fatalf("CheckBlobStore() error = %v, want wrapped health check failure", err)
	}
}

func TestPlugin_NewBlobStore_InvalidRoot(t *testing.T) {
	p := &Plugin{}
	_, err := p.NewBlobStore(context.Background(), nil, plugin.BlobStoreOptions{RootDir: "   "})
	if err == nil || !strings.Contains(err.Error(), "diskblob: invalid root_dir") {
		t.Fatalf("NewBlobStore() error = %v, want invalid root_dir error", err)
	}
}
