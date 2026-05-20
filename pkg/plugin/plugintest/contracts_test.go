package plugintest

import (
	"context"
	"slices"
	"testing"

	"github.com/edelwud/terraci/pkg/pipeline"
	"github.com/edelwud/terraci/pkg/plugin"
)

func TestAssertBaseConfigPlugin(t *testing.T) {
	p := &contractPlugin{BasePlugin: plugin.BasePlugin[*contractConfig]{
		PluginName: "contract",
		PluginDesc: "contract plugin",
		EnableMode: plugin.EnabledExplicitly,
		DefaultCfg: func() *contractConfig {
			return &contractConfig{
				Enabled: true,
				Labels:  []string{"default"},
				Nested:  &contractNested{Name: "default"},
			}
		},
		IsEnabledFn: func(c *contractConfig) bool { return c != nil && c.Enabled },
	}}

	AssertBaseConfigPlugin[*contractConfig](t, BaseConfigPluginContract[*contractConfig]{
		Plugin: p,
		Default: &contractConfig{
			Enabled: true,
			Labels:  []string{"default"},
			Nested:  &contractNested{Name: "default"},
		},
		Configured: &contractConfig{
			Enabled: true,
			Labels:  []string{"configured"},
			Nested:  &contractNested{Name: "configured"},
		},
		Decoded: &contractConfig{
			Enabled: true,
			Labels:  []string{"decoded"},
			Nested:  &contractNested{Name: "decoded"},
		},
		Mutate: mutateContractConfig,
		Equal:  equalContractConfig,
	})
}

func TestAssertCommandBinding(t *testing.T) {
	p := &contractPlugin{BasePlugin: plugin.BasePlugin[*contractConfig]{
		PluginName: "contract",
		PluginDesc: "contract plugin",
		EnableMode: plugin.EnabledAlways,
		DefaultCfg: func() *contractConfig { return &contractConfig{} },
	}}

	AssertCommandBinding[*contractPlugin](t, CommandBindingContract[*contractPlugin]{
		Name:   "contract",
		Plugin: p,
		AssertResolved: func(tb testing.TB, got *contractPlugin) {
			tb.Helper()
			if got != p {
				tb.Fatalf("resolved plugin = %p, want %p", got, p)
			}
		},
	})
}

func TestAssertRequireEnabled(t *testing.T) {
	AssertRequireEnabled(t, RequireEnabledContract{
		Enabled:  staticEnabled(true),
		Disabled: staticEnabled(false),
		Message:  "contract plugin is disabled",
	})
}

func TestAssertRuntimeProvider(t *testing.T) {
	p := &contractRuntimePlugin{contractPlugin: contractPlugin{BasePlugin: plugin.BasePlugin[*contractConfig]{
		PluginName: "contract",
		PluginDesc: "contract plugin",
		EnableMode: plugin.EnabledAlways,
		DefaultCfg: func() *contractConfig { return &contractConfig{} },
	}}}

	AssertRuntimeProvider[*contractRuntime](t, RuntimeProviderContract[*contractRuntime]{
		Provider: p,
		AssertRuntime: func(tb testing.TB, got *contractRuntime) {
			tb.Helper()
			if got == nil || got.Name != "contract" {
				tb.Fatalf("runtime = %#v, want named runtime", got)
			}
		},
	})
}

func TestAssertPipelineContributor(t *testing.T) {
	AssertPipelineContributor(t, PipelineContributorContract{
		Contributor:      contractContributor{},
		ExpectedJobNames: []string{"first", "second"},
	})
}

type contractPlugin struct {
	plugin.BasePlugin[*contractConfig]
}

type contractConfig struct {
	Enabled bool
	Labels  []string
	Nested  *contractNested
}

type contractNested struct {
	Name string
}

func (c *contractConfig) Clone() *contractConfig {
	if c == nil {
		return nil
	}
	out := *c
	out.Labels = slices.Clone(c.Labels)
	if c.Nested != nil {
		nested := *c.Nested
		out.Nested = &nested
	}
	return &out
}

func mutateContractConfig(c *contractConfig) {
	if c == nil {
		return
	}
	c.Enabled = !c.Enabled
	if len(c.Labels) > 0 {
		c.Labels[0] = "mutated"
	}
	c.Labels = append(c.Labels, "extra")
	if c.Nested != nil {
		c.Nested.Name = "mutated"
	}
}

func equalContractConfig(got, want *contractConfig) bool {
	if got == nil || want == nil {
		return got == want
	}
	if got.Enabled != want.Enabled || !slices.Equal(got.Labels, want.Labels) {
		return false
	}
	if got.Nested == nil || want.Nested == nil {
		return got.Nested == want.Nested
	}
	return got.Nested.Name == want.Nested.Name
}

type staticEnabled bool

func (e staticEnabled) IsEnabled() bool { return bool(e) }

type contractRuntime struct {
	Name string
}

type contractRuntimePlugin struct {
	contractPlugin
}

func (p *contractRuntimePlugin) Runtime(_ context.Context, _ *plugin.AppContext) (any, error) {
	return &contractRuntime{Name: p.Name()}, nil
}

type contractContributor struct{}

func (contractContributor) Name() string        { return "contract" }
func (contractContributor) Description() string { return "contract contributor" }

func (contractContributor) PipelineContribution(*plugin.AppContext) *pipeline.Contribution {
	return &pipeline.Contribution{Jobs: []pipeline.ContributedJob{
		{Name: "first"},
		{Name: "second"},
	}}
}
