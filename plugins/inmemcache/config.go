package inmemcache

// Config controls whether the built-in in-memory cache backend is active.
type Config struct {
	Enabled bool `yaml:"enabled,omitempty" json:"enabled,omitempty" jsonschema:"description=Enable the built-in in-memory KV cache backend,default=true"`
}
