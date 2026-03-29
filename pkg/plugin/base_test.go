package plugin

import "testing"

type testConfig struct {
	Name    string
	Enabled bool
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
	if b.ConfigKey() != "testbase" {
		t.Errorf("ConfigKey() = %q, want testbase", b.ConfigKey())
	}
}

func TestBasePlugin_ConfigKey_DefaultsToName(t *testing.T) {
	b := &BasePlugin[*testConfig]{
		PluginName: "myname",
		DefaultCfg: func() *testConfig { return &testConfig{} },
	}
	if b.ConfigKey() != "myname" {
		t.Errorf("ConfigKey() = %q, want myname", b.ConfigKey())
	}
}

func TestBasePlugin_NewConfig(t *testing.T) {
	b := newTestBasePlugin(EnabledWhenConfigured, nil)
	cfg := b.NewConfig()
	tc, ok := cfg.(*testConfig)
	if !ok {
		t.Fatalf("NewConfig() returned %T, want *testConfig", cfg)
	}
	if tc.Name != "default" {
		t.Errorf("NewConfig().Name = %q, want default", tc.Name)
	}
}

func TestBasePlugin_DecodeAndSet(t *testing.T) {
	b := newTestBasePlugin(EnabledWhenConfigured, nil)

	if b.IsConfigured() {
		t.Error("should not be configured before DecodeAndSet")
	}

	err := b.DecodeAndSet(func(target any) error {
		cfg, ok := target.(**testConfig)
		if !ok {
			t.Fatal("unexpected target type")
		}
		*cfg = &testConfig{Name: "decoded", Enabled: true}
		return nil
	})
	if err != nil {
		t.Fatalf("DecodeAndSet error: %v", err)
	}

	if !b.IsConfigured() {
		t.Error("should be configured after DecodeAndSet")
	}
	if b.Config().Name != "decoded" {
		t.Errorf("Config().Name = %q, want decoded", b.Config().Name)
	}
}

func TestBasePlugin_SetTypedConfig(t *testing.T) {
	b := newTestBasePlugin(EnabledWhenConfigured, nil)
	b.SetTypedConfig(&testConfig{Name: "direct"})

	if !b.IsConfigured() {
		t.Error("should be configured after SetTypedConfig")
	}
	if b.Config().Name != "direct" {
		t.Errorf("Config().Name = %q, want direct", b.Config().Name)
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

// --- Interface satisfaction ---

func TestBasePlugin_SatisfiesConfigLoader(_ *testing.T) {
	var _ ConfigLoader = newTestBasePlugin(EnabledWhenConfigured, nil)
}
