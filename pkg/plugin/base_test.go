package plugin

import (
	"strings"
	"testing"

	"github.com/edelwud/terraci/pkg/config"
)

type testConfig struct {
	Name    string
	Enabled bool
	Labels  []string
}

func (c *testConfig) Clone() *testConfig {
	if c == nil {
		return nil
	}
	out := *c
	out.Labels = append([]string(nil), c.Labels...)
	return &out
}

func newTestBasePlugin(mode EnablePolicy, enabledFn func(*testConfig) bool) *BasePlugin[*testConfig] {
	return &BasePlugin[*testConfig]{
		PluginName: "test-base",
		PluginDesc: "A test base plugin",
		PluginKey:  "testbase",
		EnableMode: mode,
		DefaultCfg: func() *testConfig {
			return &testConfig{Name: "default"}
		},
		IsEnabledFn: enabledFn,
	}
}

func TestBasePlugin_NameDescription(t *testing.T) {
	b := newTestBasePlugin(EnabledWhenConfigured, nil)
	if b.Name() != "test-base" {
		t.Errorf("Name() = %q, want test-base", b.Name())
	}
	if b.Description() != "A test base plugin" {
		t.Errorf("Description() = %q, want A test base plugin", b.Description())
	}
}

func TestBasePlugin_ConfigKey_Custom(t *testing.T) {
	b := newTestBasePlugin(EnabledWhenConfigured, nil)
	if b.ConfigKey().String() != "testbase" {
		t.Errorf("ConfigKey() = %q, want testbase", b.ConfigKey())
	}
}

func TestBasePlugin_ConfigKey_DefaultsToName(t *testing.T) {
	b := &BasePlugin[*testConfig]{
		PluginName: "myname",
		DefaultCfg: func() *testConfig { return &testConfig{} },
	}
	if b.ConfigKey().String() != "myname" {
		t.Errorf("ConfigKey() = %q, want myname", b.ConfigKey())
	}
}

func TestBasePlugin_SchemaConfig(t *testing.T) {
	b := newTestBasePlugin(EnabledWhenConfigured, nil)
	cfg := b.SchemaConfig()
	tc, ok := cfg.(*testConfig)
	if !ok {
		t.Fatalf("SchemaConfig() returned %T, want *testConfig", cfg)
	}
	if tc.Name != "default" {
		t.Errorf("SchemaConfig().Name = %q, want default", tc.Name)
	}

	tc.Name = "mutated"
	again := b.SchemaConfig().(*testConfig)
	if again.Name != "default" {
		t.Errorf("SchemaConfig() leaked mutation: Name = %q, want default", again.Name)
	}
}

func TestBasePlugin_DecodeAndSet(t *testing.T) {
	b := newTestBasePlugin(EnabledWhenConfigured, nil)

	if b.IsConfigured() {
		t.Error("should not be configured before DecodeAndSet")
	}

	doc := extensionDocument(t, "testbase", &testConfig{Name: "decoded", Enabled: true})
	err := b.DecodeAndSet(doc)
	if err != nil {
		t.Fatalf("DecodeAndSet error: %v", err)
	}

	if !b.IsConfigured() {
		t.Error("should be configured after DecodeAndSet")
	}
	if b.Config().Name != "decoded" {
		t.Errorf("Config().Name = %q, want decoded", b.Config().Name)
	}

	cfg := b.Config()
	cfg.Name = "mutated"
	if b.Config().Name != "decoded" {
		t.Errorf("Config() leaked mutation: Name = %q, want decoded", b.Config().Name)
	}
}

func TestBasePlugin_SetTypedConfig(t *testing.T) {
	b := newTestBasePlugin(EnabledWhenConfigured, nil)
	cfg := &testConfig{Name: "direct", Labels: []string{"a"}}
	b.SetTypedConfig(cfg)
	cfg.Name = "mutated"
	cfg.Labels[0] = "mutated"

	if !b.IsConfigured() {
		t.Error("should be configured after SetTypedConfig")
	}
	if b.Config().Name != "direct" {
		t.Errorf("Config().Name = %q, want direct", b.Config().Name)
	}
	if got := b.Config().Labels[0]; got != "a" {
		t.Errorf("Config().Labels[0] = %q, want a", got)
	}
}

func TestBasePlugin_Reset(t *testing.T) {
	b := newTestBasePlugin(EnabledWhenConfigured, nil)
	b.SetTypedConfig(&testConfig{Name: "set"})

	b.Reset()

	if b.IsConfigured() {
		t.Error("should not be configured after Reset")
	}
	if b.Config() != nil {
		t.Error("Config() should be nil after Reset")
	}
}

// --- EnablePolicy tests ---

func TestBasePlugin_EnabledWhenConfigured(t *testing.T) {
	b := newTestBasePlugin(EnabledWhenConfigured, nil)

	if b.IsEnabled() {
		t.Error("should not be enabled before config")
	}

	b.SetTypedConfig(&testConfig{})
	if !b.IsEnabled() {
		t.Error("should be enabled after config")
	}
}

func TestBasePlugin_EnabledExplicitly(t *testing.T) {
	b := newTestBasePlugin(EnabledExplicitly, func(cfg *testConfig) bool {
		return cfg != nil && cfg.Enabled
	})

	if b.IsEnabled() {
		t.Error("should not be enabled before config")
	}

	b.SetTypedConfig(&testConfig{Enabled: false})
	if b.IsEnabled() {
		t.Error("should not be enabled when Enabled=false")
	}

	b.SetTypedConfig(&testConfig{Enabled: true})
	if !b.IsEnabled() {
		t.Error("should be enabled when Enabled=true")
	}
}

func TestBasePlugin_EnabledExplicitly_NoFn(t *testing.T) {
	b := newTestBasePlugin(EnabledExplicitly, nil)
	b.SetTypedConfig(&testConfig{Enabled: true})

	if b.IsEnabled() {
		t.Error("should not be enabled when no IsEnabledFn")
	}
}

func TestBasePlugin_EnabledByDefault(t *testing.T) {
	b := newTestBasePlugin(EnabledByDefault, func(cfg *testConfig) bool {
		return cfg == nil || cfg.Enabled
	})

	// Before config: enabled by default
	if !b.IsEnabled() {
		t.Error("should be enabled by default before config")
	}

	// After config with Enabled=true
	b.SetTypedConfig(&testConfig{Enabled: true})
	if !b.IsEnabled() {
		t.Error("should be enabled when Enabled=true")
	}

	// After config with Enabled=false
	b.SetTypedConfig(&testConfig{Enabled: false})
	if b.IsEnabled() {
		t.Error("should not be enabled when Enabled=false")
	}
}

func TestBasePlugin_EnabledByDefault_NoFn(t *testing.T) {
	b := newTestBasePlugin(EnabledByDefault, nil)

	if !b.IsEnabled() {
		t.Error("should be enabled by default before config")
	}

	b.SetTypedConfig(&testConfig{})
	if !b.IsEnabled() {
		t.Error("should be enabled by default when no IsEnabledFn")
	}
}

func TestBasePlugin_EnabledAlways(t *testing.T) {
	b := newTestBasePlugin(EnabledAlways, nil)

	if !b.IsEnabled() {
		t.Error("should always be enabled")
	}

	b.SetTypedConfig(&testConfig{Enabled: false})
	if !b.IsEnabled() {
		t.Error("should always be enabled regardless of config")
	}
}

func TestBasePlugin_ValidateRejectsInvalidConfigKey(t *testing.T) {
	b := &BasePlugin[*testConfig]{
		PluginName: "bad",
		PluginKey:  "bad.key",
		DefaultCfg: func() *testConfig { return &testConfig{} },
	}
	err := b.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want invalid config key")
	}
	if !strings.Contains(err.Error(), "config key") {
		t.Fatalf("Validate() error = %v, want config key context", err)
	}
}

// --- Interface satisfaction ---

func TestBasePlugin_SatisfiesConfigLoader(_ *testing.T) {
	var _ ConfigLoader = newTestBasePlugin(EnabledWhenConfigured, nil)
}

func extensionDocument(t *testing.T, key string, value any) config.ExtensionDocument {
	t.Helper()
	extensionValue, err := config.NewExtensionValue(key, value)
	if err != nil {
		t.Fatalf("NewExtensionValue() error = %v", err)
	}
	cfg, err := config.Build(config.BuildOptions{
		Extensions: mustExtensionSet(t, extensionValue),
	})
	if err != nil {
		t.Fatalf("config.Build() error = %v", err)
	}
	doc, ok := cfg.Extension(config.MustExtensionKey(key))
	if !ok {
		t.Fatalf("Extension(%q) missing", key)
	}
	return doc
}

func mustExtensionSet(t *testing.T, values ...config.ExtensionValue) config.ExtensionSet {
	t.Helper()
	set, err := config.NewExtensionSet(values...)
	if err != nil {
		t.Fatalf("NewExtensionSet() error = %v", err)
	}
	return set
}
