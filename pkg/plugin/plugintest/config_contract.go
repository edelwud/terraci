package plugintest

import (
	"testing"

	"github.com/edelwud/terraci/pkg/config"
	"github.com/edelwud/terraci/pkg/plugin"
)

// BaseConfigPlugin is the public surface implemented by plugin.BasePlugin[C].
// Contract tests use this interface instead of depending on the concrete
// embedded type, so third-party plugins can wrap BasePlugin in their own
// plugin struct and still reuse the assertions.
type BaseConfigPlugin[C plugin.ConfigCloner[C]] interface {
	plugin.ConfigLoader
	Config() C
	SetTypedConfig(C)
}

// BaseConfigPluginContract describes the fixtures needed to verify the
// immutable-config contract for a BasePlugin-backed plugin.
type BaseConfigPluginContract[C plugin.ConfigCloner[C]] struct {
	Plugin     BaseConfigPlugin[C]
	Default    C
	Configured C
	Decoded    C
	Mutate     func(C)
	Equal      func(got, want C) bool
}

// AssertBaseConfigPlugin verifies the canonical config behavior external
// plugin authors rely on: configs are Clone()able, ConfigDefinition returns a
// valid schema definition, and Config/SetTypedConfig/DecodeAndSet do not leak
// mutable state.
func AssertBaseConfigPlugin[C plugin.ConfigCloner[C]](tb testing.TB, c BaseConfigPluginContract[C]) {
	tb.Helper()
	if c.Plugin == nil {
		tb.Fatal("Plugin is nil")
	}
	if c.Mutate == nil {
		tb.Fatal("Mutate is nil")
	}
	if c.Equal == nil {
		tb.Fatal("Equal is nil")
	}

	definition, err := c.Plugin.ConfigDefinition()
	if err != nil {
		tb.Fatalf("ConfigDefinition() error = %v", err)
	}
	if definition.Key().String() == "" {
		tb.Fatal("ConfigDefinition().Key() is empty")
	}
	set, err := config.NewExtensionDefinitionSet(definition)
	if err != nil {
		tb.Fatalf("NewExtensionDefinitionSet() error = %v", err)
	}
	if _, err := config.GenerateJSONSchema(set); err != nil {
		tb.Fatalf("GenerateJSONSchema() error = %v", err)
	}

	configuredWant := c.Configured.Clone()
	c.Plugin.SetTypedConfig(c.Configured)
	c.Mutate(c.Configured)
	assertConfigEqual(tb, "Config() after SetTypedConfig", c.Plugin.Config(), configuredWant, c.Equal)
	gotConfigured := c.Plugin.Config()
	c.Mutate(gotConfigured)
	assertConfigEqual(tb, "Config() after mutating returned config", c.Plugin.Config(), configuredWant, c.Equal)

	doc := configDocument(tb, definition.Key(), c.Decoded)
	decodedWant := c.Default.Clone()
	if err := doc.Decode(&decodedWant); err != nil {
		tb.Fatalf("ExtensionDocument.Decode() error = %v", err)
	}
	if err := c.Plugin.DecodeAndSet(doc); err != nil {
		tb.Fatalf("DecodeAndSet() error = %v", err)
	}
	c.Mutate(c.Decoded)
	assertConfigEqual(tb, "Config() after DecodeAndSet", c.Plugin.Config(), decodedWant, c.Equal)
	gotDecoded := c.Plugin.Config()
	c.Mutate(gotDecoded)
	assertConfigEqual(tb, "Config() after mutating decoded return value", c.Plugin.Config(), decodedWant, c.Equal)
}

func configDocument(tb testing.TB, key config.ExtensionKey, value any) config.ExtensionDocument {
	tb.Helper()
	extensionValue, err := config.NewExtensionValue(key, value)
	if err != nil {
		tb.Fatalf("NewExtensionValue() error = %v", err)
	}
	set, err := config.NewExtensionValueSet(extensionValue)
	if err != nil {
		tb.Fatalf("NewExtensionValueSet() error = %v", err)
	}
	cfg, err := config.Build(config.BuildOptions{Extensions: set})
	if err != nil {
		tb.Fatalf("config.Build() error = %v", err)
	}
	doc, ok := cfg.Extension(key)
	if !ok {
		tb.Fatalf("Extension(%q) missing", key.String())
	}
	return doc
}

func assertConfigEqual[C plugin.ConfigCloner[C]](tb testing.TB, label string, got, want C, equal func(C, C) bool) {
	tb.Helper()
	if !equal(got, want) {
		tb.Fatalf("%s = %#v, want %#v", label, got, want)
	}
}
