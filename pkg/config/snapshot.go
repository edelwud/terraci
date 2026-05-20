package config

import "go.yaml.in/yaml/v4"

// Snapshot is an immutable view of a TerraCi Config. Accessors return values
// or defensive copies so callers cannot mutate the captured configuration.
type Snapshot struct {
	cfg *Config
}

// NewSnapshot captures cfg as an immutable value. Later mutations to cfg do
// not affect the snapshot.
func NewSnapshot(cfg *Config) Snapshot {
	return Snapshot{cfg: cfg.Clone()}
}

// Snapshot returns an immutable view of c.
func (c *Config) Snapshot() Snapshot {
	return NewSnapshot(c)
}

// Present reports whether the snapshot contains a configuration.
func (s Snapshot) Present() bool {
	return s.cfg != nil
}

// MutableCopy returns a deep mutable copy of the captured configuration.
func (s Snapshot) MutableCopy() *Config {
	return s.cfg.Clone()
}

// ServiceDir returns the configured service directory.
func (s Snapshot) ServiceDir() string {
	if s.cfg == nil {
		return ""
	}
	return s.cfg.ServiceDir
}

// Execution returns a defensive copy of execution settings.
func (s Snapshot) Execution() ExecutionConfig {
	if s.cfg == nil {
		return ExecutionConfig{}
	}
	return s.cfg.Execution.clone()
}

// Structure returns a defensive copy of structure settings.
func (s Snapshot) Structure() StructureConfig {
	if s.cfg == nil {
		return StructureConfig{}
	}
	return s.cfg.Structure.clone()
}

// Exclude returns a defensive copy of exclude patterns.
func (s Snapshot) Exclude() []string {
	if s.cfg == nil {
		return nil
	}
	return append([]string(nil), s.cfg.Exclude...)
}

// Include returns a defensive copy of include patterns.
func (s Snapshot) Include() []string {
	if s.cfg == nil {
		return nil
	}
	return append([]string(nil), s.cfg.Include...)
}

// LibraryModules returns a defensive copy of library module configuration.
func (s Snapshot) LibraryModules() *LibraryModulesConfig {
	if s.cfg == nil {
		return nil
	}
	return s.cfg.LibraryModules.clone()
}

// Extensions returns a defensive copy of raw extension YAML sections.
func (s Snapshot) Extensions() map[string]yaml.Node {
	if s.cfg == nil {
		return nil
	}
	return cloneYAMLNodeMap(s.cfg.Extensions)
}

// Extension decodes the named extension's configuration section into target.
// Returns nil if the snapshot is empty or the section is missing.
func (s Snapshot) Extension(key string, target any) error {
	if s.cfg == nil {
		return nil
	}
	return s.cfg.Extension(key, target)
}
