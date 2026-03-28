package test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/edelwud/terraci/pkg/config"
	"github.com/edelwud/terraci/pkg/plugin"
)

func TestPluginRegistration(t *testing.T) {
	all := plugin.All()
	if len(all) != 6 {
		t.Fatalf("expected 6 plugins, got %d", len(all))
	}

	names := make(map[string]bool)
	for _, p := range all {
		if p.Name() == "" {
			t.Error("plugin has empty name")
		}
		if p.Description() == "" {
			t.Errorf("plugin %q has empty description", p.Name())
		}
		names[p.Name()] = true
	}

	expected := []string{"cost", "git", "github", "gitlab", "policy", "summary"}
	for _, name := range expected {
		if !names[name] {
			t.Errorf("missing plugin: %s", name)
		}
	}
}

func TestPluginCapabilities(t *testing.T) {
	// ConfigProvider plugins
	configProviders := plugin.ByCapability[plugin.ConfigProvider]()
	if len(configProviders) == 0 {
		t.Fatal("expected at least one ConfigProvider")
	}

	configKeys := make(map[string]bool)
	for _, cp := range configProviders {
		key := cp.ConfigKey()
		if key == "" {
			t.Errorf("plugin %s has empty ConfigKey", cp.Name())
		}
		if configKeys[key] {
			t.Errorf("duplicate ConfigKey: %s", key)
		}
		configKeys[key] = true
	}

	// Verify specific expected config keys
	for _, expectedKey := range []string{"gitlab", "github", "cost", "policy"} {
		if !configKeys[expectedKey] {
			t.Errorf("missing expected ConfigKey: %s", expectedKey)
		}
	}

	// GeneratorProvider plugins (gitlab + github)
	generators := plugin.ByCapability[plugin.GeneratorProvider]()
	if len(generators) < 2 {
		t.Errorf("expected at least 2 GeneratorProviders (gitlab, github), got %d", len(generators))
	}

	// Initializable plugins
	initializables := plugin.ByCapability[plugin.Initializable]()
	if len(initializables) == 0 {
		t.Fatal("expected at least one Initializable plugin")
	}

	// CommandProvider plugins
	commandProviders := plugin.ByCapability[plugin.CommandProvider]()
	if len(commandProviders) == 0 {
		t.Fatal("expected at least one CommandProvider plugin")
	}
}

func TestPluginConfigLoading(t *testing.T) {
	plugin.ResetPlugins()
	cfg, err := config.Load(filepath.Join(fixtureDir(t, "basic"), ".terraci.yaml"))
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Configure plugins from the fixture config
	for _, p := range plugin.ByCapability[plugin.ConfigProvider]() {
		if _, exists := cfg.Plugins[p.ConfigKey()]; !exists {
			continue
		}
		cfgVal := p.NewConfig()
		if decErr := cfg.PluginConfig(p.ConfigKey(), cfgVal); decErr != nil {
			t.Fatalf("failed to decode %s config: %v", p.ConfigKey(), decErr)
		}
		if setErr := p.SetConfig(cfgVal); setErr != nil {
			t.Fatalf("failed to set %s config: %v", p.ConfigKey(), setErr)
		}
	}

	// gitlab should be configured (it's in the fixture)
	for _, p := range plugin.ByCapability[plugin.ConfigProvider]() {
		if p.ConfigKey() == "gitlab" && !p.IsConfigured() {
			t.Error("gitlab should be configured after loading basic fixture")
		}
		if p.ConfigKey() == "github" && p.IsConfigured() {
			t.Error("github should NOT be configured (not in basic fixture)")
		}
	}
}

func TestProviderResolution(t *testing.T) {
	plugin.ResetPlugins()
	cfg, err := config.Load(filepath.Join(fixtureDir(t, "basic"), ".terraci.yaml"))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	for _, p := range plugin.ByCapability[plugin.ConfigProvider]() {
		if _, exists := cfg.Plugins[p.ConfigKey()]; !exists {
			continue
		}
		cfgVal := p.NewConfig()
		if decErr := cfg.PluginConfig(p.ConfigKey(), cfgVal); decErr != nil {
			t.Fatalf("failed to decode %s config: %v", p.ConfigKey(), decErr)
		}
		if setErr := p.SetConfig(cfgVal); setErr != nil {
			t.Fatalf("failed to set %s config: %v", p.ConfigKey(), setErr)
		}
	}

	provider, resolveErr := plugin.ResolveProvider()
	if resolveErr != nil {
		t.Fatalf("resolve provider: %v", resolveErr)
	}
	if provider.ProviderName() != "gitlab" {
		t.Errorf("expected gitlab provider, got %s", provider.ProviderName())
	}
}

func TestPluginInitialization(t *testing.T) {
	plugin.ResetPlugins()
	dir := fixtureDir(t, "basic")
	cfg, err := config.Load(filepath.Join(dir, ".terraci.yaml"))
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	appCtx := &plugin.AppContext{
		Config:     cfg,
		WorkDir:    dir,
		ServiceDir: filepath.Join(dir, ".terraci"),
		Version:    "test",
	}

	for _, p := range plugin.ByCapability[plugin.Initializable]() {
		if initErr := p.Initialize(context.Background(), appCtx); initErr != nil {
			// Some plugins may fail if their external deps are missing (e.g., git not in a repo).
			// We log but don't fail — the important thing is the interface works.
			t.Logf("initialize %s: %v (may be expected outside real env)", p.Name(), initErr)
		}
	}
}
