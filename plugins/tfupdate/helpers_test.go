package tfupdate

import (
	"context"
	"testing"
	"time"

	"github.com/edelwud/terraci/pkg/cache/blobcache"
	"github.com/edelwud/terraci/pkg/cache/blobcache/blobtest"
	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/pkg/plugin/plugintest"
	"github.com/edelwud/terraci/pkg/plugin/registry"
	tfupdateengine "github.com/edelwud/terraci/plugins/tfupdate/internal"
	tfregistry "github.com/edelwud/terraci/plugins/tfupdate/internal/registry"
	"github.com/edelwud/terraci/plugins/tfupdate/internal/registrymeta"
	"github.com/edelwud/terraci/plugins/tfupdate/internal/sourceaddr"
)

// mockRegistry implements tfregistry.Client for testing.
type mockRegistry struct {
	moduleVersions    map[string][]string                      // key: "ns/name/provider"
	providerVersions  map[string][]string                      // key: "ns/type"
	providerPlatforms map[string][]string                      // key: "ns/type@version"
	providerPackages  map[string]*registrymeta.ProviderPackage // key: "ns/type@version/platform"
	moduleErr         error
	providerErr       error
}

func (m *mockRegistry) ModuleVersions(_ context.Context, address sourceaddr.ModuleAddress) ([]string, error) {
	if m.moduleErr != nil {
		return nil, m.moduleErr
	}
	return m.moduleVersions[address.Namespace+"/"+address.Name+"/"+address.Provider], nil
}

func (m *mockRegistry) ModuleProviderDeps(_ context.Context, _ sourceaddr.ModuleAddress, _ string) ([]registrymeta.ModuleProviderDep, error) {
	return nil, nil
}

func (m *mockRegistry) ProviderVersions(_ context.Context, address sourceaddr.ProviderAddress) ([]string, error) {
	if m.providerErr != nil {
		return nil, m.providerErr
	}
	return m.providerVersions[address.Namespace+"/"+address.Type], nil
}

func (m *mockRegistry) ProviderPlatforms(_ context.Context, address sourceaddr.ProviderAddress, version string) ([]string, error) {
	if m.providerErr != nil {
		return nil, m.providerErr
	}
	return append([]string(nil), m.providerPlatforms[address.Namespace+"/"+address.Type+"@"+version]...), nil
}

func (m *mockRegistry) ProviderPackage(_ context.Context, address sourceaddr.ProviderAddress, version, platform string) (*registrymeta.ProviderPackage, error) {
	if m.providerErr != nil {
		return nil, m.providerErr
	}
	pkg := m.providerPackages[address.Namespace+"/"+address.Type+"@"+version+"/"+platform]
	if pkg == nil {
		return nil, nil
	}
	copyPkg := *pkg
	return &copyPkg, nil
}

// newTestPlugin creates a fresh Plugin with the same configuration as the one
// registered in init(), but without touching the global plugin registry.
func newTestPlugin(t *testing.T) *Plugin {
	t.Helper()
	p := &Plugin{
		BasePlugin: plugin.BasePlugin[*tfupdateengine.UpdateConfig]{
			PluginName: "tfupdate",
			PluginDesc: "Terraform dependency resolver and lock synchronizer",
			EnableMode: plugin.EnabledExplicitly,
			DefaultCfg: func() *tfupdateengine.UpdateConfig {
				return &tfupdateengine.UpdateConfig{
					Target: tfupdateengine.TargetAll,
					Policy: tfupdateengine.UpdatePolicy{Bump: tfupdateengine.BumpMinor},
				}
			},
			IsEnabledFn: func(cfg *tfupdateengine.UpdateConfig) bool {
				return cfg != nil && cfg.Enabled
			},
		},
	}
	t.Cleanup(p.Reset)
	return p
}

// enablePlugin configures the plugin with the given config, marking it as configured.
func enablePlugin(t *testing.T, p *Plugin, cfg *tfupdateengine.UpdateConfig) {
	t.Helper()
	if cfg != nil && cfg.Enabled && cfg.BumpPolicy() == "" {
		cfg.Policy.Bump = tfupdateengine.BumpMinor
	}
	p.SetTypedConfig(cfg)
}

func useMockRegistry(p *Plugin, reg tfregistry.Client) {
	p.registryFactory = func() tfregistry.Client { return reg }
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
	if p.cache == nil {
		p.cache = stubKVCache{}
	}
	return p.cache, nil
}

type testBlobStoreProvider struct {
	testPlugin
	store blobcache.Store
}

func (p *testBlobStoreProvider) NewBlobStore(context.Context, *plugin.AppContext, plugin.BlobStoreOptions) (blobcache.Store, error) {
	if p.store == nil {
		p.store = blobtest.NewMemoryStore("tfupdate-test")
	}
	return p.store, nil
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

func defaultKVCacheProvider() plugin.Plugin {
	return &testKVCacheProvider{
		testPlugin: testPlugin{name: "inmemcache", desc: "test in-memory KV cache backend"},
		cache:      stubKVCache{},
	}
}

func defaultBlobStoreProvider() plugin.Plugin {
	return &testBlobStoreProvider{
		testPlugin: testPlugin{name: "diskblob", desc: "test blob store backend"},
		store:      blobtest.NewMemoryStore("tfupdate-test"),
	}
}

func newTestBackendRegistry(t *testing.T, factories ...registry.Factory) *registry.Registry {
	t.Helper()
	all := make([]registry.Factory, 0, 2+len(factories))
	all = append(all, defaultKVCacheProvider, defaultBlobStoreProvider)
	all = append(all, factories...)
	return plugintest.NewRegistry(t, all...)
}

type commandTestResolver struct {
	plugin.NoopResolver
	backends *registry.Registry
}

func (r commandTestResolver) ResolveKVCacheProvider(name string, configPathHint ...string) (plugin.KVCacheProvider, error) {
	return r.backends.ResolveKVCacheProvider(name, configPathHint...)
}

func (r commandTestResolver) ResolveBlobStoreProvider(name string, configPathHint ...string) (plugin.BlobStoreProvider, error) {
	return r.backends.ResolveBlobStoreProvider(name, configPathHint...)
}

// newTestAppContext creates a minimal AppContext suitable for plugin testing.
func newTestAppContext(t *testing.T, workDir string) *plugin.AppContext {
	t.Helper()
	return plugintest.NewAppContextWithResolver(t, workDir, newTestBackendRegistry(t))
}

func newTestCommandAppContext(t *testing.T, workDir string, _ *Plugin) *plugin.AppContext {
	t.Helper()
	return plugintest.NewAppContextWithResolver(t, workDir, commandTestResolver{
		backends: newTestBackendRegistry(t),
	})
}

func newTestAppContextWithResolver(t *testing.T, workDir string, resolver plugin.Resolver) *plugin.AppContext {
	return plugintest.NewAppContextWithResolver(t, workDir, resolver)
}
