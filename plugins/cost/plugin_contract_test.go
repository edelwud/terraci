package cost

import (
	"maps"
	"testing"

	"github.com/edelwud/terraci/pkg/plugin/initwiz"
	"github.com/edelwud/terraci/pkg/plugin/plugintest"
	"github.com/edelwud/terraci/plugins/cost/internal/model"
)

func TestPlugin_SDKContracts(t *testing.T) {
	t.Run("config", func(t *testing.T) {
		p := newTestPlugin(t)
		plugintest.AssertBaseConfigPlugin[*model.CostConfig](t, plugintest.BaseConfigPluginContract[*model.CostConfig]{
			Plugin:  p,
			Default: &model.CostConfig{},
			Configured: &model.CostConfig{
				Providers: model.CostProvidersConfig{"aws": {Enabled: true}},
				BlobCache: &model.BlobCacheConfig{
					Backend:   "diskblob",
					Namespace: "cost/test",
					TTL:       "1h",
				},
			},
			Decoded: &model.CostConfig{
				Providers: model.CostProvidersConfig{"aws": {Enabled: true}, "gcp": {Enabled: false}},
				BlobCache: &model.BlobCacheConfig{
					Backend:   "memory",
					Namespace: "decoded",
					TTL:       "2h",
				},
			},
			Mutate: mutateCostConfig,
			Equal:  equalCostConfig,
		})
	})

	t.Run("command binding", func(t *testing.T) {
		p := newTestPlugin(t)
		plugintest.AssertCommandBinding[*Plugin](t, plugintest.CommandBindingContract[*Plugin]{
			Name:   pluginName,
			Plugin: p,
			AssertResolved: func(tb testing.TB, got *Plugin) {
				tb.Helper()
				if got != p {
					tb.Fatalf("resolved plugin = %p, want %p", got, p)
				}
			},
		})
	})

	t.Run("require enabled", func(t *testing.T) {
		enabled := newTestPlugin(t)
		enablePlugin(t, enabled, &model.CostConfig{Providers: model.CostProvidersConfig{"aws": {Enabled: true}}})
		disabled := newTestPlugin(t)
		plugintest.AssertRequireEnabled(t, plugintest.RequireEnabledContract{
			Enabled:  enabled,
			Disabled: disabled,
			Message:  "cost disabled",
		})
	})

	t.Run("preflight", func(t *testing.T) {
		p := newTestPlugin(t)
		enablePlugin(t, p, &model.CostConfig{Providers: model.CostProvidersConfig{"aws": {Enabled: true}}})
		plugintest.AssertPreflightable(t, plugintest.PreflightableContract{
			Plugin:     p,
			AppContext: newTestAppContext(t, t.TempDir()),
		})
	})

	t.Run("init contributor", func(t *testing.T) {
		state := initwiz.NewStateMap()
		providerEnabledKey("aws").Set(state, true)
		plugintest.AssertInitContributor(t, plugintest.InitContributorContract{
			Contributor:        newTestPlugin(t),
			State:              state,
			ExpectedPluginKey:  pluginName,
			ExpectContribution: true,
			DecodeTarget:       &model.CostConfig{},
		})
	})
}

func mutateCostConfig(c *model.CostConfig) {
	if c == nil {
		return
	}
	if c.Providers == nil {
		c.Providers = model.CostProvidersConfig{}
	}
	c.Providers["aws"] = model.ProviderConfig{Enabled: false}
	c.Providers["new"] = model.ProviderConfig{Enabled: true}
	if c.BlobCache != nil {
		c.BlobCache.Namespace = "mutated"
	}
}

func equalCostConfig(got, want *model.CostConfig) bool {
	if got == nil || want == nil {
		return got == want
	}
	if !maps.Equal(got.Providers, want.Providers) {
		return false
	}
	if got.BlobCache == nil || want.BlobCache == nil {
		return got.BlobCache == want.BlobCache
	}
	return *got.BlobCache == *want.BlobCache
}
