package runflow

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

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

func (p *configPlugin) ConfigKey() string { return p.name }
func (p *configPlugin) NewConfig() any    { return &testConfig{} }
func (p *configPlugin) DecodeAndSet(decode func(target any) error) error {
	var cfg testConfig
	if err := decode(&cfg); err != nil {
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
		SkipConfig:  true,
	})
	if err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}
	if result.AppContext == nil {
		t.Fatal("AppContext is nil")
	}
	if result.Config.Present() {
		t.Fatal("Config snapshot should be empty for skip-config command")
	}
	if result.AppContext.Resolver() == nil {
		t.Fatal("Resolver is nil")
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
	if !result.Config.Present() || result.Loaded == nil {
		t.Fatal("loaded config not captured")
	}
	if result.AppContext.Config().Structure().Pattern == "" {
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
		CommandName:   "validate",
		ConfigPath:    filepath.Join(dir, ".terraci.yaml"),
		WorkDir:       dir,
		LogLevel:      "info",
		SkipPreflight: true,
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
	if len(result.AppContext.PipelineContributions()) != 1 {
		t.Fatalf("PipelineContributions len = %d, want 1", len(result.AppContext.PipelineContributions()))
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
				return testPlugin{name: "bare"}
			})
		},
	})
	req := Request{CommandName: "version", WorkDir: t.TempDir(), SkipConfig: true}

	first, err := flow.Prepare(context.Background(), req)
	if err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}
	second, err := flow.Prepare(context.Background(), req)
	if err != nil {
		t.Fatalf("Prepare() second error = %v", err)
	}
	if first.Registry == second.Registry {
		t.Fatal("registry should be fresh for each command run")
	}
	if first.Reports != second.Reports {
		t.Fatal("report store should persist across command runs")
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
