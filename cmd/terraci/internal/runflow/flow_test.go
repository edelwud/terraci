package runflow

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"

	"github.com/edelwud/terraci/pkg/config"
	"github.com/edelwud/terraci/pkg/pipeline"
	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/pkg/plugin/registry"
)

type testConfig struct {
	Enabled bool `yaml:"enabled"`
}

type testPlugin struct {
	name string
}

func (p testPlugin) Name() string        { return p.name }
func (p testPlugin) Description() string { return p.name }

type configPlugin struct {
	testPlugin
	configured bool
	enabled    bool
}

func (p *configPlugin) ConfigKey() config.ExtensionKey {
	return config.MustExtensionKey(p.name)
}
func (p *configPlugin) SchemaConfig() any { return &testConfig{} }
func (p *configPlugin) DecodeAndSet(doc config.ExtensionDocument) error {
	var cfg testConfig
	if err := doc.Decode(&cfg); err != nil {
		return err
	}
	p.configured = true
	p.enabled = cfg.Enabled
	return nil
}
func (p *configPlugin) IsConfigured() bool { return p.configured }
func (p *configPlugin) IsEnabled() bool    { return p.enabled }

type preflightPlugin struct {
	*configPlugin
	err    error
	called bool
}

func (p *preflightPlugin) Preflight(context.Context, *plugin.AppContext) error {
	p.called = true
	return p.err
}

type contributorPlugin struct {
	testPlugin
	err    error
	called bool
}

func (p *contributorPlugin) PipelineContribution(*plugin.AppContext) (*pipeline.Contribution, error) {
	p.called = true
	if p.err != nil {
		return nil, p.err
	}
	job, err := pipeline.NewPluginCommandJob(pipeline.PluginCommandJobOptions{
		Name:     p.name + "-job",
		Commands: []string{"terraci " + p.name},
	})
	if err != nil {
		return nil, err
	}
	return pipeline.NewContribution(job)
}

type commandPlugin struct {
	testPlugin
}

func (p commandPlugin) CommandSpecs() ([]plugin.CommandSpec, error) {
	cmd, err := plugin.NewCommandSpec(plugin.CommandSpecOptions{
		Use:  p.name,
		RunE: func(*cobra.Command, []string) error { return nil },
	})
	if err != nil {
		return nil, err
	}
	return []plugin.CommandSpec{cmd}, nil
}

func TestPrepareSkipConfigBindsContextWithoutConfigLoad(t *testing.T) {
	t.Parallel()

	result, err := New(Options{
		RegistryFactory: func() *registry.Registry {
			return registry.NewFromFactories(func() plugin.Plugin {
				return testPlugin{name: "bare"}
			})
		},
		Version: "test",
	}).Prepare(context.Background(), Request{
		CommandName: "version",
		WorkDir:     filepath.Join(t.TempDir(), "missing"),
		LogLevel:    "info",
		Policy:      CommandPolicy{SkipConfig: true},
	})
	if err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}
	if result.AppContext() == nil {
		t.Fatal("AppContext is nil")
	}
	if result.Config().Present() {
		t.Fatal("Config snapshot should be empty for skip-config command")
	}
	if result.AppContext().CIResolver() == nil {
		t.Fatal("CIResolver is nil")
	}
	if got, err := FromContext(result.Context()); err != nil || got != result {
		t.Fatalf("FromContext() = %p, %v; want prepared", got, err)
	}
}

func TestPrepareLoadsConfigAndDecodesPluginConfig(t *testing.T) {
	t.Parallel()

	cfgPlugin := &configPlugin{testPlugin: testPlugin{name: "fake"}}
	dir := writeRunConfig(t, `structure:
  pattern: "{service}/{environment}/{region}/{module}"
extensions:
  fake:
    enabled: true
`)

	result, err := New(Options{
		RegistryFactory: func() *registry.Registry {
			return registry.NewFromFactories(func() plugin.Plugin { return cfgPlugin })
		},
		Version: "test",
	}).Prepare(context.Background(), Request{
		CommandName: "generate",
		ConfigPath:  filepath.Join(dir, ".terraci.yaml"),
		WorkDir:     dir,
		LogLevel:    "info",
	})
	if err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}
	if !cfgPlugin.configured {
		t.Fatal("plugin config was not decoded")
	}
	if !result.Config().Present() || result.LoadedConfig() == nil {
		t.Fatal("loaded config not captured")
	}
	if result.AppContext().Config().Structure().Pattern == "" {
		t.Fatal("AppContext config snapshot is empty")
	}
}

func TestPrepareSkipPreflightStillCollectsContributions(t *testing.T) {
	t.Parallel()

	errPreflight := errors.New("preflight should be skipped")
	preflight := &preflightPlugin{
		configPlugin: &configPlugin{testPlugin: testPlugin{name: "preflight"}},
		err:          errPreflight,
	}
	contributor := &contributorPlugin{testPlugin: testPlugin{name: "contrib"}}
	dir := writeRunConfig(t, `structure:
  pattern: "{service}/{environment}/{region}/{module}"
extensions:
  preflight:
    enabled: true
`)

	result, err := New(Options{
		RegistryFactory: func() *registry.Registry {
			return registry.NewFromFactories(
				func() plugin.Plugin { return preflight },
				func() plugin.Plugin { return contributor },
			)
		},
		Version: "test",
	}).Prepare(context.Background(), Request{
		CommandName: "validate",
		ConfigPath:  filepath.Join(dir, ".terraci.yaml"),
		WorkDir:     dir,
		LogLevel:    "info",
		Policy:      CommandPolicy{SkipPreflight: true},
	})
	if err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}
	if preflight.called {
		t.Fatal("preflight was called despite SkipPreflight")
	}
	if !contributor.called {
		t.Fatal("pipeline contribution was not collected")
	}
	if len(result.PipelineContributions()) != 1 {
		t.Fatalf("Prepared.PipelineContributions len = %d, want 1", len(result.PipelineContributions()))
	}
	cmd := &cobra.Command{}
	cmd.SetContext(result.Context())
	cmdCtx, _, err := plugin.CommandPlugin[*contributorPlugin](cmd, "contrib")
	if err != nil {
		t.Fatalf("CommandPlugin() error = %v", err)
	}
	if len(cmdCtx.PipelineContributions()) != 1 {
		t.Fatalf("CommandContext.PipelineContributions len = %d, want 1", len(cmdCtx.PipelineContributions()))
	}
}

func TestPreparePreflightErrorWrapsPluginError(t *testing.T) {
	t.Parallel()

	errPreflight := errors.New("boom")
	preflight := &preflightPlugin{
		configPlugin: &configPlugin{testPlugin: testPlugin{name: "preflight"}},
		err:          errPreflight,
	}
	dir := writeRunConfig(t, `structure:
  pattern: "{service}/{environment}/{region}/{module}"
extensions:
  preflight:
    enabled: true
`)

	_, err := New(Options{
		RegistryFactory: func() *registry.Registry {
			return registry.NewFromFactories(func() plugin.Plugin { return preflight })
		},
	}).Prepare(context.Background(), Request{
		CommandName: "generate",
		ConfigPath:  filepath.Join(dir, ".terraci.yaml"),
		WorkDir:     dir,
		LogLevel:    "info",
	})
	if !errors.Is(err, errPreflight) {
		t.Fatalf("Prepare() error = %v, want wrapping %v", err, errPreflight)
	}
}

func TestPrepareContributionErrorPreservesTypedError(t *testing.T) {
	t.Parallel()

	errContribution := errors.New("bad contribution")
	contributor := &contributorPlugin{testPlugin: testPlugin{name: "contrib"}, err: errContribution}
	dir := writeRunConfig(t, `structure:
  pattern: "{service}/{environment}/{region}/{module}"
`)

	_, err := New(Options{
		RegistryFactory: func() *registry.Registry {
			return registry.NewFromFactories(func() plugin.Plugin { return contributor })
		},
	}).Prepare(context.Background(), Request{
		CommandName: "generate",
		ConfigPath:  filepath.Join(dir, ".terraci.yaml"),
		WorkDir:     dir,
		LogLevel:    "info",
	})
	var contributionErr *plugin.PipelineContributionError
	if !errors.As(err, &contributionErr) {
		t.Fatalf("Prepare() error = %T, want PipelineContributionError", err)
	}
	if !errors.Is(err, errContribution) {
		t.Fatalf("Prepare() error = %v, want wrapping %v", err, errContribution)
	}
}

func TestPrepareKeepsReportStoreAndRefreshesRegistry(t *testing.T) {
	t.Parallel()

	flow := New(Options{
		RegistryFactory: func() *registry.Registry {
			return registry.NewFromFactories(func() plugin.Plugin {
				return &commandPlugin{testPlugin: testPlugin{name: "bare"}}
			})
		},
	})
	req := Request{CommandName: "version", WorkDir: t.TempDir(), Policy: CommandPolicy{SkipConfig: true}}

	first, err := flow.Prepare(context.Background(), req)
	if err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}
	second, err := flow.Prepare(context.Background(), req)
	if err != nil {
		t.Fatalf("Prepare() second error = %v", err)
	}
	firstCmd := &cobra.Command{}
	firstCmd.SetContext(first.Context())
	_, firstPlugin, err := plugin.CommandPlugin[*commandPlugin](firstCmd, "bare")
	if err != nil {
		t.Fatalf("CommandPlugin(first) error = %v", err)
	}
	secondCmd := &cobra.Command{}
	secondCmd.SetContext(second.Context())
	_, secondPlugin, err := plugin.CommandPlugin[*commandPlugin](secondCmd, "bare")
	if err != nil {
		t.Fatalf("CommandPlugin(second) error = %v", err)
	}
	if firstPlugin == secondPlugin {
		t.Fatal("command-scoped plugin should be fresh for each command run")
	}
	if first.Reports() != second.Reports() {
		t.Fatal("report store should persist across command runs")
	}
}

func TestCommandPolicyRoundTrip(t *testing.T) {
	t.Parallel()

	policy := CommandPolicy{SkipConfig: true, SkipPreflight: true}
	cmd := MarkCommand(&cobra.Command{}, policy)
	if got := PolicyFromCommand(cmd); got != policy {
		t.Fatalf("PolicyFromCommand() = %#v, want %#v", got, policy)
	}
}

func TestPluginCommandsUsesFreshRegistry(t *testing.T) {
	t.Parallel()

	commands, err := PluginCommands(func() *registry.Registry {
		return registry.NewFromFactories(func() plugin.Plugin {
			return commandPlugin{testPlugin: testPlugin{name: "hello"}}
		})
	})
	if err != nil {
		t.Fatalf("PluginCommands() error = %v", err)
	}
	if len(commands) != 1 || commands[0].Use != "hello" {
		t.Fatalf("PluginCommands() = %#v, want hello command", commands)
	}
}

func writeRunConfig(t *testing.T, raw string) string {
	t.Helper()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".terraci.yaml"), []byte(raw), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return dir
}
