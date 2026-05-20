package config

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"go.yaml.in/yaml/v4"
)

var extensionKeyPattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

// BuildOptions describes a typed config construction request.
type BuildOptions struct {
	Pattern    string
	Execution  *ExecutionConfig
	Extensions ExtensionSet
}

// ExtensionValue is a validated extension config section ready to be stored as
// an opaque YAML node in Config.Extensions.
type ExtensionValue struct {
	key  string
	node yaml.Node
}

// ExtensionSet is a duplicate-free set of extension config values.
type ExtensionSet struct {
	values []ExtensionValue
}

// NewExtensionValue encodes value as the config section for key.
func NewExtensionValue(key string, value any) (ExtensionValue, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return ExtensionValue{}, errors.New("extension key is required")
	}
	if !extensionKeyPattern.MatchString(key) {
		return ExtensionValue{}, fmt.Errorf("extension key %q is invalid; use letters, digits, underscore, or hyphen", key)
	}
	if value == nil {
		return ExtensionValue{}, fmt.Errorf("extension %q config is nil", key)
	}

	node, err := encodeYAMLNode(value)
	if err != nil {
		return ExtensionValue{}, fmt.Errorf("encode extension %q config: %w", key, err)
	}
	return ExtensionValue{key: key, node: node}, nil
}

// Key returns the extension config key.
func (v ExtensionValue) Key() string {
	return v.key
}

// Decode decodes the extension config into target.
func (v ExtensionValue) Decode(target any) error {
	return v.node.Decode(target)
}

// Node returns a defensive copy of the encoded YAML node.
func (v ExtensionValue) Node() yaml.Node {
	return cloneYAMLNode(v.node)
}

// Clone returns a defensive copy of v.
func (v ExtensionValue) Clone() ExtensionValue {
	return v.clone()
}

func (v ExtensionValue) clone() ExtensionValue {
	v.node = cloneYAMLNode(v.node)
	return v
}

// NewExtensionSet builds a duplicate-free extension set.
func NewExtensionSet(values ...ExtensionValue) (ExtensionSet, error) {
	seen := make(map[string]struct{}, len(values))
	cloned := make([]ExtensionValue, 0, len(values))
	for i := range values {
		value := values[i]
		if value.key == "" {
			return ExtensionSet{}, fmt.Errorf("extensions[%d]: extension key is required", i)
		}
		if _, exists := seen[value.key]; exists {
			return ExtensionSet{}, fmt.Errorf("duplicate extension %q", value.key)
		}
		seen[value.key] = struct{}{}
		cloned = append(cloned, value.clone())
	}
	return ExtensionSet{values: cloned}, nil
}

// Values returns defensive copies of extension values in declaration order.
func (s ExtensionSet) Values() []ExtensionValue {
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
	if cfg.Extensions == nil {
		cfg.Extensions = make(map[string]yaml.Node)
	}
	cfg.Extensions[value.key] = cloneYAMLNode(value.node)
}
