package tfupdate

import (
	"context"
	"testing"
	"time"

	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/pkg/plugin/plugintest"
	"github.com/edelwud/terraci/pkg/plugin/registry"
	tfupdateengine "github.com/edelwud/terraci/plugins/tfupdate/internal"
)

func TestPlugin_Name(t *testing.T) {
	p := newTestPlugin(t)
	if p.Name() != "tfupdate" {
		t.Errorf("Name() = %q, want %q", p.Name(), "tfupdate")
	}
	if p.Description() == "" {
		t.Error("Description() is empty")
	}
}

func TestPlugin_EnablePolicy(t *testing.T) {
	tests := []struct {
		name           string
		cfg            *tfupdateengine.UpdateConfig
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
			cfg:            &tfupdateengine.UpdateConfig{Enabled: false},
			setCfg:         true,
			wantConfigured: true,
			wantEnabled:    false,
		},
		{
			name:           "config set, enabled=true",
			cfg:            &tfupdateengine.UpdateConfig{Enabled: true},
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
	if p.registryFactory != nil {
		t.Error("registry factory should be nil when plugin is not configured")
	}
}

func TestPlugin_Preflight_ConfiguredButDisabled(t *testing.T) {
	p := newTestPlugin(t)
	enablePlugin(t, p, &tfupdateengine.UpdateConfig{Enabled: false})
	appCtx := newTestAppContext(t, t.TempDir())

	if err := p.Preflight(context.Background(), appCtx); err != nil {
		t.Fatalf("Preflight() error = %v", err)
	}
	if p.registryFactory != nil {
		t.Error("registry factory should be nil when plugin is configured but disabled")
	}
}

func TestPlugin_Preflight_Enabled(t *testing.T) {
	p := newTestPlugin(t)
	enablePlugin(t, p, &tfupdateengine.UpdateConfig{Enabled: true})
	appCtx := newTestAppContext(t, t.TempDir())

	if err := p.Preflight(context.Background(), appCtx); err != nil {
		t.Fatalf("Preflight() error = %v", err)
	}
	if p.registryFactory != nil {
		t.Fatal("registry factory should remain nil after preflight; runtime is lazy")
	}
}

func TestPlugin_Preflight_InvalidConfig(t *testing.T) {
	p := newTestPlugin(t)
	enablePlugin(t, p, &tfupdateengine.UpdateConfig{
		Enabled: true,
		Target:  "invalid-target",
	})
	appCtx := newTestAppContext(t, t.TempDir())

	if err := p.Preflight(context.Background(), appCtx); err == nil {
		t.Fatal("Preflight() error = nil, want invalid configuration error")
	}
}

func TestPlugin_Reset(t *testing.T) {
	p := newTestPlugin(t)
	enablePlugin(t, p, &tfupdateengine.UpdateConfig{Enabled: true})
	useMockRegistry(p, &mockRegistry{})

	p.Reset()

	if p.IsConfigured() {
		t.Error("IsConfigured() should be false after Reset")
	}
	if p.registryFactory != nil {
		t.Error("registry factory should be nil after Reset")
	}
}

func TestPlugin_Runtime_CreatesRegistryLazily(t *testing.T) {
	p := newTestPlugin(t)
	enablePlugin(t, p, &tfupdateengine.UpdateConfig{Enabled: true})
	useMockRegistry(p, &mockRegistry{})

	runtime := plugintest.MustRuntime[*updateRuntime](t, p, newTestAppContext(t, t.TempDir()))
	if runtime.registry == nil {
		t.Fatal("runtime.registry should not be nil")
	}
}

func TestPlugin_Runtime_DefaultsToInmemcache(t *testing.T) {
	p := newTestPlugin(t)
	enablePlugin(t, p, &tfupdateengine.UpdateConfig{Enabled: true})
	useMockRegistry(p, &mockRegistry{})

	runtime := plugintest.MustRuntime[*updateRuntime](t, p, newTestAppContext(t, t.TempDir()))
	if runtime.registry == nil {
		t.Fatal("runtime.registry should not be nil")
	}

	got, err := runtime.registry.ModuleVersions(context.Background(), "registry.terraform.io", "hashicorp", "consul", "aws")
	if err != nil {
		t.Fatalf("ModuleVersions() error = %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("ModuleVersions() = %v, want empty slice for mock response", got)
	}
}

func TestPlugin_Runtime_UnknownCacheBackend(t *testing.T) {
	p := newTestPlugin(t)
	enablePlugin(t, p, &tfupdateengine.UpdateConfig{
		Enabled: true,
		Cache: &tfupdateengine.CacheConfig{
			Backend: "missing-backend",
		},
	})
	useMockRegistry(p, &mockRegistry{})

	if _, err := p.Runtime(context.Background(), newTestAppContext(t, t.TempDir())); err == nil {
		t.Fatal("Runtime() error = nil, want missing backend error")
	}
}

func TestPlugin_Runtime_DefaultBackendStableWithAdditionalProvider(t *testing.T) {
	alt := &testKVCacheProvider{
		testPlugin: testPlugin{
			name: "alt-cache-provider",
			desc: "extra cache backend",
		},
		cache: stubKVCache{},
	}
	registry.Register(alt)

	p := newTestPlugin(t)
	enablePlugin(t, p, &tfupdateengine.UpdateConfig{Enabled: true})
	useMockRegistry(p, &mockRegistry{
		moduleVersions: map[string][]string{
			"hashicorp/consul/aws": {"1.0.0"},
		},
	})

	runtime := plugintest.MustRuntime[*updateRuntime](t, p, newTestAppContext(t, t.TempDir()))
	got, err := runtime.registry.ModuleVersions(context.Background(), "registry.terraform.io", "hashicorp", "consul", "aws")
	if err != nil {
		t.Fatalf("ModuleVersions() error = %v", err)
	}
	if len(got) != 1 || got[0] != "1.0.0" {
		t.Fatalf("ModuleVersions() = %v, want [1.0.0]", got)
	}
	if alt.calls != 0 {
		t.Fatalf("alt cache provider should not be used by default, got %d calls", alt.calls)
	}
}

type testPlugin struct {
	name string
	desc string
}

func (p testPlugin) Name() string        { return p.name }
func (p testPlugin) Description() string { return p.desc }

type testKVCacheProvider struct {
	testPlugin
	cache plugin.KVCache
	calls int
}

func (p *testKVCacheProvider) NewKVCache(context.Context, *plugin.AppContext) (plugin.KVCache, error) {
	p.calls++
	return p.cache, nil
}

type stubKVCache struct{}

func (stubKVCache) Get(context.Context, string, string) (value []byte, found bool, err error) {
	return nil, false, nil
}
func (stubKVCache) Set(context.Context, string, string, []byte, time.Duration) error {
	return nil
}
func (stubKVCache) Delete(context.Context, string, string) error  { return nil }
func (stubKVCache) DeleteNamespace(context.Context, string) error { return nil }
