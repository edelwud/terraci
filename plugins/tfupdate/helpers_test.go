package tfupdate

import (
	"context"
	"testing"

	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/pkg/plugin/plugintest"
	"github.com/edelwud/terraci/pkg/plugin/registry"
	_ "github.com/edelwud/terraci/plugins/diskblob"
	_ "github.com/edelwud/terraci/plugins/inmemcache"
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
					Bump:   tfupdateengine.BumpMinor,
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

type commandTestResolver struct {
	plugintest.NoopResolver
	plugin   plugin.Plugin
	backends *registry.Registry
}

func (r commandTestResolver) All() []plugin.Plugin {
	all := r.backends.All()
	if r.plugin != nil {
		return append([]plugin.Plugin{r.plugin}, all...)
	}
	return all
}

func (r commandTestResolver) GetPlugin(name string) (plugin.Plugin, bool) {
	if r.plugin != nil && r.plugin.Name() == name {
		return r.plugin, true
	}
	return r.backends.GetPlugin(name)
}

func (r commandTestResolver) ResolveKVCacheProvider(name string) (plugin.KVCacheProvider, error) {
	return r.backends.ResolveKVCacheProvider(name)
}

func (r commandTestResolver) ResolveBlobStoreProvider(name string) (plugin.BlobStoreProvider, error) {
	return r.backends.ResolveBlobStoreProvider(name)
}

// newTestAppContext creates a minimal AppContext suitable for plugin testing.
func newTestAppContext(t *testing.T, workDir string) *plugin.AppContext {
	return plugintest.NewAppContext(t, workDir)
}

func newTestCommandAppContext(t *testing.T, workDir string, p *Plugin) *plugin.AppContext {
	t.Helper()
	return plugintest.NewAppContextWithResolver(t, workDir, commandTestResolver{
		plugin:   p,
		backends: registry.New(),
	})
}

func newTestAppContextWithResolver(t *testing.T, workDir string, resolver plugin.Resolver) *plugin.AppContext {
	return plugintest.NewAppContextWithResolver(t, workDir, resolver)
}
