package cost

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/pkg/plugin/plugintest"
	"github.com/edelwud/terraci/pkg/plugin/registry"
	"github.com/edelwud/terraci/plugins/cost/internal/model"
)

func TestPlugin_Name(t *testing.T) {
	p := newTestPlugin(t)
	if p.Name() != "cost" {
		t.Errorf("Name() = %q, want %q", p.Name(), "cost")
	}
	if p.Description() == "" {
		t.Error("Description() is empty")
	}
}

func TestPlugin_EnablePolicy(t *testing.T) {
	tests := []struct {
		name           string
		cfg            *model.CostConfig
		setCfg         bool
		wantConfigured bool
		wantEnabled    bool
	}{
		{
			name:           "no config set",
			setCfg:         false,
			wantConfigured: false,
			wantEnabled:    false,
		},
		{
			name:           "config set, enabled=false",
			cfg:            &model.CostConfig{Enabled: false},
			setCfg:         true,
			wantConfigured: true,
			wantEnabled:    false,
		},
		{
			name:           "config set, enabled=true",
			cfg:            &model.CostConfig{Enabled: true},
			setCfg:         true,
			wantConfigured: true,
			wantEnabled:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := newTestPlugin(t)
			if tt.setCfg {
				enablePlugin(t, p, tt.cfg)
			}
			if got := p.IsConfigured(); got != tt.wantConfigured {
				t.Errorf("IsConfigured() = %v, want %v", got, tt.wantConfigured)
			}
			if got := p.IsEnabled(); got != tt.wantEnabled {
				t.Errorf("IsEnabled() = %v, want %v", got, tt.wantEnabled)
			}
		})
	}
}

func TestPlugin_Preflight_Disabled(t *testing.T) {
	p := newTestPlugin(t)
	appCtx := newTestAppContext(t, t.TempDir())

	if err := p.Preflight(context.Background(), appCtx); err != nil {
		t.Fatalf("Preflight() error = %v", err)
	}
}

func TestPlugin_Preflight_ConfiguredButDisabled(t *testing.T) {
	p := newTestPlugin(t)
	enablePlugin(t, p, &model.CostConfig{Enabled: false})
	appCtx := newTestAppContext(t, t.TempDir())

	if err := p.Preflight(context.Background(), appCtx); err != nil {
		t.Fatalf("Preflight() error = %v", err)
	}
}

func TestPlugin_Preflight_Enabled(t *testing.T) {
	p := newTestPlugin(t)
	enablePlugin(t, p, &model.CostConfig{
		Enabled: true,
	})
	appCtx := newTestAppContext(t, t.TempDir())

	if err := p.Preflight(context.Background(), appCtx); err != nil {
		t.Fatalf("Preflight() error = %v", err)
	}
}

func TestPlugin_Preflight_InvalidTTL(t *testing.T) {
	p := newTestPlugin(t)
	enablePlugin(t, p, &model.CostConfig{
		Enabled:  true,
		CacheTTL: "not-a-duration",
	})
	appCtx := newTestAppContext(t, t.TempDir())

	err := p.Preflight(context.Background(), appCtx)
	if err == nil {
		t.Fatal("Preflight() error = nil, want invalid configuration error")
	}
	if got := err.Error(); got == "" || got == "invalid cost configuration" {
		t.Fatalf("Preflight() error = %q, want actionable validation error", got)
	}
}

func TestPlugin_Reset(t *testing.T) {
	p := newTestPlugin(t)
	enablePlugin(t, p, &model.CostConfig{
		Enabled: true,
	})
	appCtx := newTestAppContext(t, t.TempDir())

	if err := p.Preflight(context.Background(), appCtx); err != nil {
		t.Fatalf("Preflight() error = %v", err)
	}

	p.Reset()

	if p.IsConfigured() {
		t.Error("IsConfigured() should be false after Reset")
	}
}

func TestPlugin_Runtime_CreatesEstimator(t *testing.T) {
	p := newTestPlugin(t)
	enablePlugin(t, p, &model.CostConfig{
		Enabled: true,
	})

	runtime := plugintest.MustRuntime[*costRuntime](t, p, newTestAppContext(t, t.TempDir()))
	if runtime.estimator == nil {
		t.Fatal("runtime.estimator should not be nil")
	}
}

func TestPlugin_Runtime_DefaultsToDiskblob(t *testing.T) {
	p := newTestPlugin(t)
	enablePlugin(t, p, &model.CostConfig{
		Enabled: true,
		Providers: model.CostProvidersConfig{
			AWS: &model.ProviderConfig{Enabled: true},
		},
	})

	appCtx := newTestAppContext(t, t.TempDir())
	runtime := plugintest.MustRuntime[*costRuntime](t, p, appCtx)
	if runtime.estimator == nil {
		t.Fatal("runtime.estimator should not be nil")
	}
	wantCacheDir := appCtx.ServiceDir() + "/blobs"
	if runtime.estimator.CacheDir() != wantCacheDir {
		t.Fatalf("CacheDir() = %q, want %q", runtime.estimator.CacheDir(), wantCacheDir)
	}
}

func TestPlugin_Runtime_RejectsLegacyCacheDir(t *testing.T) {
	p := newTestPlugin(t)
	enablePlugin(t, p, &model.CostConfig{
		Enabled:  true,
		CacheDir: t.TempDir(),
		Providers: model.CostProvidersConfig{
			AWS: &model.ProviderConfig{Enabled: true},
		},
	})

	_, err := p.Runtime(context.Background(), newTestAppContext(t, t.TempDir()))
	if err == nil || !strings.Contains(err.Error(), "plugins.cost.cache_dir is no longer supported") {
		t.Fatalf("Runtime() error = %v, want unsupported cache_dir error", err)
	}
}

func TestPlugin_Runtime_UnknownBlobBackend(t *testing.T) {
	p := newTestPlugin(t)
	enablePlugin(t, p, &model.CostConfig{
		Enabled: true,
		BlobCache: &model.BlobCacheConfig{
			Backend: "missing-backend",
		},
		Providers: model.CostProvidersConfig{
			AWS: &model.ProviderConfig{Enabled: true},
		},
	})

	if _, err := p.Runtime(context.Background(), newTestAppContext(t, t.TempDir())); err == nil {
		t.Fatal("Runtime() error = nil, want missing backend error")
	}
}

func TestPlugin_Runtime_UsesBlobStoreDiagnostics(t *testing.T) {
	providerName := "blob-dx-cost"
	registerTestBlobStoreProvider(t, &testBlobStoreProvider{
		name: providerName,
		store: testBlobStoreWithDiagnostics{
			info: plugin.BlobStoreInfo{
				Backend:                 "diagnostic-blob",
				Root:                    "/tmp/blob-cache",
				SupportsList:            true,
				SupportsStream:          true,
				SupportsDeleteNamespace: true,
			},
		},
	})

	p := newTestPlugin(t)
	enablePlugin(t, p, &model.CostConfig{
		Enabled: true,
		BlobCache: &model.BlobCacheConfig{
			Backend: providerName,
		},
		Providers: model.CostProvidersConfig{
			AWS: &model.ProviderConfig{Enabled: true},
		},
	})

	runtime := plugintest.MustRuntime[*costRuntime](t, p, newTestAppContext(t, t.TempDir()))
	if runtime.estimator == nil {
		t.Fatal("runtime.estimator should not be nil")
	}
	if runtime.estimator.CacheDir() != "/tmp/blob-cache" {
		t.Fatalf("CacheDir() = %q, want %q", runtime.estimator.CacheDir(), "/tmp/blob-cache")
	}
}

func TestPlugin_Runtime_UsesBlobStoreFallbackDiagnostics(t *testing.T) {
	providerName := "blob-legacy-root-cost"
	registerTestBlobStoreProvider(t, &testBlobStoreProvider{
		name: providerName,
		store: testBlobStoreWithInspector{
			root: "/tmp/legacy-cache",
		},
	})

	p := newTestPlugin(t)
	enablePlugin(t, p, &model.CostConfig{
		Enabled: true,
		BlobCache: &model.BlobCacheConfig{
			Backend: providerName,
		},
		Providers: model.CostProvidersConfig{
			AWS: &model.ProviderConfig{Enabled: true},
		},
	})

	runtime := plugintest.MustRuntime[*costRuntime](t, p, newTestAppContext(t, t.TempDir()))
	if runtime.estimator.CacheDir() != "/tmp/legacy-cache" {
		t.Fatalf("CacheDir() = %q, want %q", runtime.estimator.CacheDir(), "/tmp/legacy-cache")
	}
}

func TestPlugin_Runtime_BlobStoreFallbackWithoutDiagnostics(t *testing.T) {
	providerName := "blob-legacy-cost"
	registerTestBlobStoreProvider(t, &testBlobStoreProvider{
		name:  providerName,
		store: plainTestBlobStore{},
	})

	p := newTestPlugin(t)
	enablePlugin(t, p, &model.CostConfig{
		Enabled: true,
		BlobCache: &model.BlobCacheConfig{
			Backend: providerName,
		},
		Providers: model.CostProvidersConfig{
			AWS: &model.ProviderConfig{Enabled: true},
		},
	})

	runtime := plugintest.MustRuntime[*costRuntime](t, p, newTestAppContext(t, t.TempDir()))
	if runtime.estimator == nil {
		t.Fatal("runtime.estimator should not be nil")
	}
	if runtime.estimator.CacheDir() != "" {
		t.Fatalf("CacheDir() = %q, want empty root for plain blob store", runtime.estimator.CacheDir())
	}
}

func TestPlugin_Runtime_HealthCheckFailure(t *testing.T) {
	providerName := "blob-healthfail-cost"
	registerTestBlobStoreProvider(t, &testBlobStoreProvider{
		name: providerName,
		store: testBlobStoreWithDiagnostics{
			healthErr: errors.New("root unavailable"),
		},
	})

	p := newTestPlugin(t)
	enablePlugin(t, p, &model.CostConfig{
		Enabled: true,
		BlobCache: &model.BlobCacheConfig{
			Backend: providerName,
		},
		Providers: model.CostProvidersConfig{
			AWS: &model.ProviderConfig{Enabled: true},
		},
	})

	_, err := p.Runtime(context.Background(), newTestAppContext(t, t.TempDir()))
	if err == nil || err.Error() == "" {
		t.Fatal("Runtime() error = nil, want health check failure")
	}
	if got := err.Error(); got == "" || !strings.Contains(got, "check blob backend") || !strings.Contains(got, "health check failed") {
		t.Fatalf("Runtime() error = %q, want actionable backend health error", got)
	}
}

type testBlobStoreProvider struct {
	name  string
	store plugin.BlobStore
}

func (p *testBlobStoreProvider) Name() string        { return p.name }
func (p *testBlobStoreProvider) Description() string { return "test blob store provider" }
func (p *testBlobStoreProvider) NewBlobStore(context.Context, *plugin.AppContext) (plugin.BlobStore, error) {
	return p.store, nil
}

type plainTestBlobStore struct{}

func (plainTestBlobStore) Get(context.Context, string, string) (data []byte, ok bool, meta plugin.BlobMeta, err error) {
	return nil, false, plugin.BlobMeta{}, nil
}
func (plainTestBlobStore) Put(context.Context, string, string, []byte, plugin.PutBlobOptions) (plugin.BlobMeta, error) {
	return plugin.BlobMeta{}, nil
}
func (plainTestBlobStore) Open(context.Context, string, string) (io.ReadCloser, bool, plugin.BlobMeta, error) {
	return nil, false, plugin.BlobMeta{}, nil
}
func (plainTestBlobStore) PutStream(context.Context, string, string, io.Reader, plugin.PutBlobOptions) (plugin.BlobMeta, error) {
	return plugin.BlobMeta{}, nil
}
func (plainTestBlobStore) Delete(context.Context, string, string) error              { return nil }
func (plainTestBlobStore) DeleteNamespace(context.Context, string) error             { return nil }
func (plainTestBlobStore) List(context.Context, string) ([]plugin.BlobObject, error) { return nil, nil }

type testBlobStoreWithInspector struct {
	plainTestBlobStore
	root string
}

func (s testBlobStoreWithInspector) BlobStoreRootDir() string {
	return s.root
}

type testBlobStoreWithDiagnostics struct {
	plainTestBlobStore
	info      plugin.BlobStoreInfo
	healthErr error
}

func (s testBlobStoreWithDiagnostics) DescribeBlobStore() plugin.BlobStoreInfo {
	return s.info
}

func (s testBlobStoreWithDiagnostics) CheckBlobStore(context.Context) error {
	if s.healthErr != nil {
		return errors.New("diskblob: health check failed: root unavailable")
	}
	return nil
}

func registerTestBlobStoreProvider(t *testing.T, provider *testBlobStoreProvider) {
	t.Helper()
	registry.Register(provider)
}
