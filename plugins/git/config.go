package git

// Config controls the git change-detection plugin.
type Config struct {
	// AutoUnshallow tells the plugin to deepen a shallow clone via
	// `git fetch --unshallow` before computing a diff. When false (default),
	// the plugin returns ErrShallowRepository so the user explicitly opts
	// into mutating their working copy.
	AutoUnshallow bool `yaml:"auto_unshallow,omitempty" json:"auto_unshallow,omitempty" jsonschema:"description=Automatically run 'git fetch --unshallow' when a shallow clone is detected during change detection,default=false"`
}
