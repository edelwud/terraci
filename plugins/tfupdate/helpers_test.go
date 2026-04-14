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
	"github.com/edelwud/terraci/plugins/tfupdate/internal/registrymeta"
)

// mockRegistry implements tfupdateengine.RegistryClient for testing.
type mockRegistry struct {
	moduleVersions    map[string][]string                      // key: "ns/name/provider"
	providerVersions  map[string][]string                      // key: "ns/type"
	providerPlatforms map[string][]string                      // key: "ns/type@version"
	providerPackages  map[string]*registrymeta.ProviderPackage // key: "ns/type@version/platform"
	moduleErr         error
	providerErr       error
}

func (m *mockRegistry) ModuleVersions(_ context.Context, _, ns, name, provider string) ([]string, error) {
	if m.moduleErr != nil {
		return nil, m.moduleErr
	}
	return m.moduleVersions[ns+"/"+name+"/"+provider], nil
}

func (m *mockRegistry) ModuleProviderDeps(_ context.Context, _, _, _, _, _ string) ([]registrymeta.ModuleProviderDep, error) {
	return nil, nil
}

func (m *mockRegistry) ProviderVersions(_ context.Context, _, ns, typeName string) ([]string, error) {
	if m.providerErr != nil {
		return nil, m.providerErr
	}
	return m.providerVersions[ns+"/"+typeName], nil
}

func (m *mockRegistry) ProviderPlatforms(_ context.Context, _, ns, typeName, version string) ([]string, error) {
	if m.providerErr != nil {
		return nil, m.providerErr
	}
	return append([]string(nil), m.providerPlatforms[ns+"/"+typeName+"@"+version]...), nil
}

func (m *mockRegistry) ProviderPackage(_ context.Context, _, ns, typeName, version, platform string) (*registrymeta.ProviderPackage, error) {
	if m.providerErr != nil {
		return nil, m.providerErr
	}
	pkg := m.providerPackages[ns+"/"+typeName+"@"+version+"/"+platform]
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
	registry.ResetPlugins()
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

func useMockRegistry(p *Plugin, reg tfupdateengine.RegistryClient) {
	p.registryFactory = func() tfupdateengine.RegistryClient { return reg }
}

// newTestAppContext creates a minimal AppContext suitable for plugin testing.
func newTestAppContext(t *testing.T, workDir string) *plugin.AppContext {
	return plugintest.NewAppContext(t, workDir)
}
