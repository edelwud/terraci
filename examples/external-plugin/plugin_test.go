package hello

import (
	"testing"

	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/pkg/plugin/plugintest"
)

func TestPlugin_SDKContracts(t *testing.T) {
	t.Run("config", func(t *testing.T) {
		p := newTestPlugin()
		plugintest.AssertBaseConfigPlugin[*Config](t, plugintest.BaseConfigPluginContract[*Config]{
			Plugin:     p,
			Default:    &Config{},
			Configured: &Config{Greeting: "hello"},
			Decoded:    &Config{Greeting: "decoded"},
			Mutate: func(c *Config) {
				if c != nil {
					c.Greeting = "mutated"
				}
			},
			Equal: func(got, want *Config) bool {
				if got == nil || want == nil {
					return got == want
				}
				return got.Greeting == want.Greeting
			},
		})
	})

	t.Run("command binding", func(t *testing.T) {
		p := newTestPlugin()
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
}

func newTestPlugin() *Plugin {
	return &Plugin{BasePlugin: plugin.BasePlugin[*Config]{
		PluginName: pluginName,
		PluginDesc: "Example external command plugin",
		EnableMode: plugin.EnabledAlways,
		DefaultCfg: func() *Config { return &Config{} },
	}}
}
