package config

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"go.yaml.in/yaml/v4"
)

var extensionKeyPattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

// ExtensionKey is a validated key for one section under "extensions:".
type ExtensionKey struct {
	value string
}

// NewExtensionKey validates and normalizes an extension config key.
func NewExtensionKey(key string) (ExtensionKey, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return ExtensionKey{}, errors.New("extension key is required")
	}
	if !extensionKeyPattern.MatchString(key) {
		return ExtensionKey{}, fmt.Errorf("extension key %q is invalid; use letters, digits, underscore, or hyphen", key)
	}
	return ExtensionKey{value: key}, nil
}

// MustExtensionKey validates key and panics on invalid package-level constants.
func MustExtensionKey(key string) ExtensionKey {
	extensionKey, err := NewExtensionKey(key)
	if err != nil {
		panic(err)
	}
	return extensionKey
}

// String returns the YAML extension key.
func (k ExtensionKey) String() string {
	return k.value
}

// BuildOptions describes a typed config construction request.
type BuildOptions struct {
	Pattern    string
	Execution  *ExecutionConfig
	Extensions ExtensionValueSet
}

// ExtensionValue is a validated extension config section ready to be stored as
// an opaque YAML node in Config.
type ExtensionValue struct {
	key  ExtensionKey
	node yaml.Node
}

// ExtensionValueSet is a duplicate-free set of extension config values.
type ExtensionValueSet struct {
	values []ExtensionValue
}

// NewExtensionValue encodes value as the config section for key.
func NewExtensionValue[C any](key ExtensionKey, value C) (ExtensionValue, error) {
	extensionKey := key
	if extensionKey.String() == "" {
		return ExtensionValue{}, errors.New("extension key is required")
	}
	if isNilValue(value) {
		return ExtensionValue{}, fmt.Errorf("extension %q config is nil", extensionKey.String())
	}

	node, err := encodeYAMLNode(value)
	if err != nil {
		return ExtensionValue{}, fmt.Errorf("encode extension %q config: %w", extensionKey.String(), err)
	}
	return ExtensionValue{key: extensionKey, node: node}, nil
}

// Key returns the validated extension config key.
func (v ExtensionValue) Key() ExtensionKey {
	return v.key
}

// Decode decodes the extension config into target.
func (v ExtensionValue) Decode(target any) error {
	return v.node.Decode(target)
}

// Clone returns a defensive copy of v.
func (v ExtensionValue) Clone() ExtensionValue {
	return v.clone()
}

func (v ExtensionValue) clone() ExtensionValue {
	v.node = cloneYAMLNode(v.node)
	return v
}

// NewExtensionValueSet builds a duplicate-free extension set sorted by key.
func NewExtensionValueSet(values ...ExtensionValue) (ExtensionValueSet, error) {
	seen := make(map[string]struct{}, len(values))
	cloned := make([]ExtensionValue, 0, len(values))
	for i := range values {
		value := values[i]
		key := value.key.String()
		if key == "" {
			return ExtensionValueSet{}, fmt.Errorf("extensions[%d]: extension key is required", i)
		}
		if _, exists := seen[key]; exists {
			return ExtensionValueSet{}, fmt.Errorf("duplicate extension %q", key)
		}
		seen[key] = struct{}{}
		cloned = append(cloned, value.clone())
	}
	sort.Slice(cloned, func(i, j int) bool {
		return cloned[i].key.String() < cloned[j].key.String()
	})
	return ExtensionValueSet{values: cloned}, nil
}

// Values returns defensive copies of extension values in deterministic key order.
func (s ExtensionValueSet) Values() []ExtensionValue {
	if len(s.values) == 0 {
		return nil
	}
	out := make([]ExtensionValue, len(s.values))
	for i := range s.values {
		out[i] = s.values[i].clone()
	}
	return out
}

// Build assembles a Config from typed options.
func Build(opts BuildOptions) (*Config, error) {
	cfg := DefaultConfig()
	if opts.Pattern != "" {
		cfg.Structure.Pattern = opts.Pattern
		if segments, err := ParsePattern(opts.Pattern); err == nil {
			cfg.Structure.Segments = segments
		}
	}
	if opts.Execution != nil {
		cfg.Execution = opts.Execution.clone()
	}
	for i := range opts.Extensions.values {
		setExtensionValue(cfg, opts.Extensions.values[i])
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}
	return cfg, nil
}

func encodeYAMLNode(value any) (yaml.Node, error) {
	data, err := yaml.Marshal(value)
	if err != nil {
		return yaml.Node{}, err
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return yaml.Node{}, err
	}
	if doc.Kind == yaml.DocumentNode && len(doc.Content) > 0 {
		return cloneYAMLNode(*doc.Content[0]), nil
	}
	return cloneYAMLNode(doc), nil
}

func setExtensionValue(cfg *Config, value ExtensionValue) {
	if cfg.extensions == nil {
		cfg.extensions = make(extensionNodeMap)
	}
	cfg.extensions[value.key.String()] = cloneYAMLNode(value.node)
}
