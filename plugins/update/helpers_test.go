package update

import (
	"context"
	"testing"

	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/pkg/plugin/plugintest"
	updateengine "github.com/edelwud/terraci/plugins/update/internal"
)

// mockRegistry implements updateengine.RegistryClient for testing.
type mockRegistry struct {
	moduleVersions   map[string][]string // key: "ns/name/provider"
	providerVersions map[string][]string // key: "ns/type"
	moduleErr        error
	providerErr      error
}

func (m *mockRegistry) ModuleVersions(_ context.Context, ns, name, provider string) ([]string, error) {
	if m.moduleErr != nil {
		return nil, m.moduleErr
	}
	return m.moduleVersions[ns+"/"+name+"/"+provider], nil
}

func (m *mockRegistry) ProviderVersions(_ context.Context, ns, typeName string) ([]string, error) {
	if m.providerErr != nil {
		return nil, m.providerErr
	}
	return m.providerVersions[ns+"/"+typeName], nil
}

// newTestPlugin creates a fresh Plugin with the same configuration as the one
// registered in init(), but without touching the global plugin registry.
func newTestPlugin(t *testing.T) *Plugin {
	t.Helper()
	p := &Plugin{
		BasePlugin: plugin.BasePlugin[*updateengine.UpdateConfig]{
			PluginName: "update",
			PluginDesc: "Terraform dependency version checker and updater",
			EnableMode: plugin.EnabledExplicitly,
			DefaultCfg: func() *updateengine.UpdateConfig {
				return &updateengine.UpdateConfig{
					Target: updateengine.TargetAll,
					Bump:   updateengine.BumpMinor,
				}
			},
			IsEnabledFn: func(cfg *updateengine.UpdateConfig) bool {
				return cfg != nil && cfg.Enabled
			},
		},
	}
	t.Cleanup(p.Reset)
	return p
}

// enablePlugin configures the plugin with the given config, marking it as configured.
func enablePlugin(t *testing.T, p *Plugin, cfg *updateengine.UpdateConfig) {
	t.Helper()
	p.SetTypedConfig(cfg)
}

func useMockRegistry(p *Plugin, reg updateengine.RegistryClient) {
	p.registryFactory = func() updateengine.RegistryClient { return reg }
}

// newTestAppContext creates a minimal AppContext suitable for plugin testing.
func newTestAppContext(t *testing.T, workDir string) *plugin.AppContext {
	return plugintest.NewAppContext(t, workDir)
}
