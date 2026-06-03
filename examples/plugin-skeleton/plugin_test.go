package skeleton

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/ci/citest"
	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/pkg/plugin/plugintest"
)

func TestPlugin_SDKContracts(t *testing.T) {
	t.Run("config", func(t *testing.T) {
		p := newTestPlugin()
		plugintest.AssertBaseConfigPlugin[*Config](t, plugintest.BaseConfigPluginContract[*Config]{
			Plugin:     p,
			Default:    &Config{Greeting: "Hello from skeleton!"},
			Configured: &Config{Enabled: true, Greeting: "hello"},
			Decoded:    &Config{Enabled: true, Greeting: "decoded"},
			Mutate: func(c *Config) {
				if c != nil {
					c.Enabled = !c.Enabled
					c.Greeting = "mutated"
				}
			},
			Equal: func(got, want *Config) bool {
				if got == nil || want == nil {
					return got == want
				}
				return got.Enabled == want.Enabled && got.Greeting == want.Greeting
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

	t.Run("command provider", func(t *testing.T) {
		plugintest.AssertCommandProvider(t, plugintest.CommandProviderContract{
			Provider:     newTestPlugin(),
			ExpectedUses: []string{pluginName},
		})
	})

	t.Run("require enabled", func(t *testing.T) {
		enabled := newTestPlugin()
		enabled.SetTypedConfig(&Config{Enabled: true})
		disabled := newTestPlugin()
		plugintest.AssertRequireEnabled(t, plugintest.RequireEnabledContract{
			Enabled:  enabled,
			Disabled: disabled,
			Message:  "skeleton disabled",
		})
	})
}

func TestRun_ProducerPublishesArtifactsAndOutput(t *testing.T) {
	store := ci.NewMemoryReportStore()
	runtime := Runtime{
		Config:     &Config{Enabled: true, Greeting: "from test"},
		WorkDir:    t.TempDir(),
		ServiceDir: t.TempDir(),
		Reports:    store,
	}

	result, err := Run(context.Background(), runtime, Request{})
	if err != nil {
		t.Fatalf("Run(producer) error = %v", err)
	}
	if result.Producer == nil {
		t.Fatal("Run(producer).Producer = nil")
	}

	report, ok := store.Get(pluginName)
	if !ok {
		t.Fatal("report was not published")
	}
	citest.AssertRenderedReportContract(t, report, citest.RenderedReportContract{
		Producer: pluginName,
		Status:   ci.ReportStatusPass,
	})

	var out bytes.Buffer
	if err := WriteOutput(&out, result); err != nil {
		t.Fatalf("WriteOutput() error = %v", err)
	}
	if !strings.Contains(out.String(), "from test") {
		t.Fatalf("output = %q, want greeting", out.String())
	}
}

func TestRun_ConsumerReadsReportContract(t *testing.T) {
	store := ci.NewMemoryReportStore()
	store.Publish(citest.MustRenderedReport(ci.RenderedReportOptions{
		Producer: "cost",
		Title:    "Cost",
		Status:   ci.ReportStatusWarn,
		Summary:  "cost summary",
		Sections: []ci.RenderedSectionOptions{{
			Title:  "Cost details",
			Blocks: []ci.RenderBlock{ci.NewTextBlock(ci.RenderText("ok"))},
		}},
	}))

	result, err := Run(context.Background(), Runtime{Reports: store}, Request{Consume: true})
	if err != nil {
		t.Fatalf("Run(consumer) error = %v", err)
	}
	if result.Consumer == nil || len(result.Consumer.Reports) != 1 {
		t.Fatalf("consumer reports = %#v, want one report", result.Consumer)
	}
	got := result.Consumer.Reports[0]
	if got.Producer != "cost" || got.Status != ci.ReportStatusWarn || len(got.Sections) != 1 {
		t.Fatalf("consumed report = %#v, want cost warn report", got)
	}
}

func newTestPlugin() *Plugin {
	return &Plugin{BasePlugin: plugin.BasePlugin[*Config]{
		PluginName: pluginName,
		PluginDesc: "Reference skeleton plugin (producer + consumer patterns)",
		EnableMode: plugin.EnabledExplicitly,
		DefaultCfg: func() *Config {
			return &Config{Greeting: "Hello from skeleton!"}
		},
		IsEnabledFn: func(c *Config) bool {
			return c != nil && c.Enabled
		},
	}}
}
