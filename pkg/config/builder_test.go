package config

import (
	"reflect"
	"testing"
)

func TestBuild_WithPattern(t *testing.T) {
	t.Parallel()

	cfg, err := Build(BuildOptions{Pattern: "{service}/{environment}/{module}"})
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Structure.Pattern != "{service}/{environment}/{module}" {
		t.Errorf("pattern = %q, want {service}/{environment}/{module}", cfg.Structure.Pattern)
	}
	if len(cfg.Structure.Segments) != 3 {
		t.Errorf("segments count = %d, want 3", len(cfg.Structure.Segments))
	}
}

func TestBuild_EmptyPattern(t *testing.T) {
	t.Parallel()

	cfg, err := Build(BuildOptions{})
	if err != nil {
		t.Fatal(err)
	}

	// Should use default pattern
	if cfg.Structure.Pattern == "" {
		t.Error("expected default pattern, got empty")
	}
}

func TestBuild_ProviderA(t *testing.T) {
	t.Parallel()

	extensions := mustExtensionValueSet(t, "provider_a", map[string]any{
		"image": map[string]any{"name": "hashicorp/terraform:1.6"},
		"mr": map[string]any{
			"comment": map[string]any{"enabled": true},
		},
	})
	execution := DefaultConfig().Execution
	execution.Binary = "terraform"
	execution.InitEnabled = true

	cfg, err := Build(BuildOptions{
		Pattern:    "{service}/{environment}/{module}",
		Execution:  &execution,
		Extensions: extensions,
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := cfg.Extension(MustExtensionKey("provider_a")); !ok {
		t.Fatal("expected provider_a in extensions")
	}
	if _, ok := cfg.Extension(MustExtensionKey("provider_b")); ok {
		t.Error("expected provider_b to be absent")
	}

	var providerACfg map[string]any
	decodeExtension(t, cfg, "provider_a", &providerACfg)
	if providerACfg["mr"] == nil {
		t.Error("mr config should be present")
	}
}

func TestBuild_ProviderB(t *testing.T) {
	t.Parallel()

	extensions := mustExtensionValueSet(t, "provider_b", map[string]any{
		"runs_on": "ubuntu-latest",
		"pr": map[string]any{
			"comment": map[string]any{},
		},
	})
	execution := DefaultConfig().Execution
	execution.Binary = "tofu"
	execution.InitEnabled = true

	cfg, err := Build(BuildOptions{Execution: &execution, Extensions: extensions})
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := cfg.Extension(MustExtensionKey("provider_b")); !ok {
		t.Fatal("expected provider_b in extensions")
	}
	if _, ok := cfg.Extension(MustExtensionKey("provider_a")); ok {
		t.Error("expected provider_a to be absent")
	}

	var providerBCfg map[string]any
	decodeExtension(t, cfg, "provider_b", &providerBCfg)
	if cfg.Execution.Binary != "tofu" {
		t.Errorf("binary = %v, want tofu", cfg.Execution.Binary)
	}
	if providerBCfg["pr"] == nil {
		t.Error("pr config should be present")
	}
}

func TestBuild_WithFeature(t *testing.T) {
	t.Parallel()

	extensions := mustExtensionValueSet(t, "feature_a", map[string]any{"enabled": true})
	execution := DefaultConfig().Execution
	execution.Binary = "terraform"

	cfg, err := Build(BuildOptions{Execution: &execution, Extensions: extensions})
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := cfg.Extension(MustExtensionKey("feature_a")); !ok {
		t.Error("expected feature_a in extensions")
	}

	var featureCfg map[string]any
	decodeExtension(t, cfg, "feature_a", &featureCfg)
	if featureCfg["enabled"] != true {
		t.Error("feature_a should be enabled")
	}
}

func TestNewExtensionKey(t *testing.T) {
	t.Parallel()

	key, err := NewExtensionKey(" feature_a ")
	if err != nil {
		t.Fatalf("NewExtensionKey() error = %v", err)
	}
	if key.String() != "feature_a" {
		t.Fatalf("NewExtensionKey().String() = %q, want feature_a", key.String())
	}
	if _, err := NewExtensionKey("bad.key"); err == nil {
		t.Fatal("NewExtensionKey() error = nil, want invalid key error")
	}
}

func TestBuild_InvalidPattern(t *testing.T) {
	t.Parallel()

	_, err := Build(BuildOptions{Pattern: "{service}/{service}"})
	if err == nil {
		t.Fatal("expected validation error for invalid pattern")
	}
}

func TestBuild_InvalidExecution(t *testing.T) {
	t.Parallel()

	execution := DefaultConfig().Execution
	execution.Binary = "terragrunt"
	_, err := Build(BuildOptions{Execution: &execution})
	if err == nil {
		t.Fatal("expected validation error for invalid execution.binary")
	}
}

func TestNewExtensionValueSet_DuplicateKey(t *testing.T) {
	t.Parallel()

	first := mustExtensionValue(t, "feature", map[string]any{"enabled": true})
	second := mustExtensionValue(t, "feature", map[string]any{"enabled": false})
	if _, err := NewExtensionValueSet(first, second); err == nil {
		t.Fatal("NewExtensionValueSet() error = nil, want duplicate key error")
	}
}

func TestNewExtensionValueSet_SortsByKey(t *testing.T) {
	t.Parallel()

	first := mustExtensionValue(t, "b_feature", map[string]any{"enabled": true})
	second := mustExtensionValue(t, "a_feature", map[string]any{"enabled": true})
	set, err := NewExtensionValueSet(first, second)
	if err != nil {
		t.Fatalf("NewExtensionValueSet() error = %v", err)
	}
	values := set.Values()
	if got := []string{values[0].Key().String(), values[1].Key().String()}; !reflect.DeepEqual(got, []string{"a_feature", "b_feature"}) {
		t.Fatalf("Values() keys = %#v, want sorted keys", got)
	}
}

func TestNewExtensionValue_InvalidInput(t *testing.T) {
	t.Parallel()

	if _, err := NewExtensionValue(ExtensionKey{}, map[string]any{"enabled": true}); err == nil {
		t.Fatal("NewExtensionValue() error = nil, want key error")
	}
	if _, err := NewExtensionValue[any](MustExtensionKey("feature"), nil); err == nil {
		t.Fatal("NewExtensionValue() error = nil, want nil config error")
	}
}

func TestExtensionValue_DecodeUsesDefensiveNode(t *testing.T) {
	t.Parallel()

	value := mustExtensionValue(t, "feature", map[string]any{"enabled": true})
	clone := value.Clone()
	value.node.Content = nil

	var decoded map[string]any
	if err := clone.Decode(&decoded); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if decoded["enabled"] != true {
		t.Fatalf("decoded = %#v, want enabled true", decoded)
	}
}

func TestExtensionValueSet_ValuesAreDefensive(t *testing.T) {
	t.Parallel()

	set := mustExtensionValueSet(t, "feature", map[string]any{"enabled": true})
	values := set.Values()
	if len(values) != 1 {
		t.Fatalf("Values() len = %d, want 1", len(values))
	}
	values[0].node.Content = nil

	cfg, err := Build(BuildOptions{Extensions: set})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	var decoded map[string]any
	decodeExtension(t, cfg, "feature", &decoded)
	if decoded["enabled"] != true {
		t.Fatalf("decoded = %#v, want enabled true", decoded)
	}
}

func decodeExtension(tb testing.TB, cfg *Config, key string, target any) {
	tb.Helper()
	doc, ok := cfg.Extension(MustExtensionKey(key))
	if !ok {
		tb.Fatalf("Extension(%q) missing", key)
	}
	if err := doc.Decode(target); err != nil {
		tb.Fatalf("Extension(%q).Decode() error = %v", key, err)
	}
}

func mustExtensionValueSet(tb testing.TB, key string, value any) ExtensionValueSet {
	tb.Helper()
	extension := mustExtensionValue(tb, key, value)
	set, err := NewExtensionValueSet(extension)
	if err != nil {
		tb.Fatalf("NewExtensionValueSet() error = %v", err)
	}
	return set
}

func mustExtensionValue(tb testing.TB, key string, value any) ExtensionValue {
	tb.Helper()
	extension, err := NewExtensionValue(MustExtensionKey(key), value)
	if err != nil {
		tb.Fatalf("NewExtensionValue() error = %v", err)
	}
	return extension
}
