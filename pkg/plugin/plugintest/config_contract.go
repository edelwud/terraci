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
// plugin authors rely on: configs are Clone()able, SchemaConfig returns a fresh
// default, and Config/SetTypedConfig/DecodeAndSet do not leak mutable state.
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

	assertConfigEqual(tb, "SchemaConfig()", asConfig[C](tb, c.Plugin.SchemaConfig()), c.Default, c.Equal)
	firstDefault := asConfig[C](tb, c.Plugin.SchemaConfig())
	c.Mutate(firstDefault)
	assertConfigEqual(tb, "SchemaConfig() after mutating prior default", asConfig[C](tb, c.Plugin.SchemaConfig()), c.Default, c.Equal)

	configuredWant := c.Configured.Clone()
	c.Plugin.SetTypedConfig(c.Configured)
	c.Mutate(c.Configured)
	assertConfigEqual(tb, "Config() after SetTypedConfig", c.Plugin.Config(), configuredWant, c.Equal)
	gotConfigured := c.Plugin.Config()
	c.Mutate(gotConfigured)
	assertConfigEqual(tb, "Config() after mutating returned config", c.Plugin.Config(), configuredWant, c.Equal)

	doc := configDocument(tb, c.Plugin.ConfigKey(), c.Decoded)
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
	extensionValue, err := config.NewExtensionValue(key.String(), value)
	if err != nil {
		tb.Fatalf("NewExtensionValue() error = %v", err)
	}
	set, err := config.NewExtensionSet(extensionValue)
	if err != nil {
		tb.Fatalf("NewExtensionSet() error = %v", err)
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

func asConfig[C plugin.ConfigCloner[C]](tb testing.TB, value any) C {
	tb.Helper()
	cfg, ok := value.(C)
	if !ok {
		var zero C
		tb.Fatalf("config type = %T, want %T", value, zero)
	}
	return cfg
}

func assertConfigEqual[C plugin.ConfigCloner[C]](tb testing.TB, label string, got, want C, equal func(C, C) bool) {
	tb.Helper()
	if !equal(got, want) {
		tb.Fatalf("%s = %#v, want %#v", label, got, want)
	}
}
