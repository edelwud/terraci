package diskblob

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/pkg/plugin/plugintest"
)

func TestStore_ContractSuite(t *testing.T) {
	plugintest.RunBlobStoreContractSuite(t, plugintest.BlobStoreContractSuite{
		Factory: func(tb testing.TB, backingID string) plugin.BlobStore {
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

func TestPlugin_NewBlobStoreWithOptions_InvalidRoot(t *testing.T) {
	p := &Plugin{}
	_, err := p.NewBlobStoreWithOptions(context.Background(), nil, plugin.BlobStoreOptions{RootDir: "   "})
	if err == nil || !strings.Contains(err.Error(), "diskblob: invalid root_dir") {
		t.Fatalf("NewBlobStoreWithOptions() error = %v, want invalid root_dir error", err)
	}
}
