// Package skeleton is an annotated reference plugin that demonstrates the
// two most common third-party plugin shapes:
//
//  1. Producer — writes a typed CI report (`{producer}-report.json`) so that
//     the built-in summary plugin can pick it up and render it in MR/PR
//     comments. The cost / policy / tfupdate built-ins follow this shape.
//
//  2. Consumer — reads other plugins' reports from the service directory.
//     The summary plugin is the canonical consumer; localexec reuses the
//     same load path locally.
//
// Both shapes are wired in this single module so an author can copy the
// file they need and ignore the rest. See report.go for the producer
// pattern and consumer.go for the consumer pattern.
//
// To build a TerraCi binary that includes this skeleton:
//
//	xterraci build --with github.com/edelwud/terraci/examples/plugin-skeleton=./examples/plugin-skeleton
//
// To use it in a project:
//
//	# .terraci.yaml
//	extensions:
//	  skeleton:
//	    enabled: true
//	    greeting: "Hello from skeleton!"
//
//	terraci skeleton             # writes skeleton-report.json
//	terraci skeleton --consume   # reads other plugins' reports
package skeleton

import (
	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/pkg/plugin/registry"
)

const pluginName = "skeleton"

// init registers the plugin factory at process startup. The framework
// validates the BasePlugin shape before storing the factory: a misconfigured
// plugin (e.g. EnabledExplicitly without IsEnabledFn) panics here, not at
// first command invocation.
func init() {
	registry.RegisterFactory(func() plugin.Plugin {
		return &Plugin{
			BasePlugin: plugin.BasePlugin[*Config]{
				PluginName: pluginName,
				PluginDesc: "Reference skeleton plugin (producer + consumer patterns)",

				// EnabledExplicitly: only run when the user opts in via
				// `enabled: true`. Use EnabledByDefault for plugins that
				// should be on unless explicitly disabled.
				EnableMode: plugin.EnabledExplicitly,

				DefaultCfg: func() *Config {
					return &Config{Greeting: "Hello from skeleton!"}
				},
				IsEnabledFn: func(c *Config) bool {
					return c != nil && c.Enabled
				},
			},
		}
	})
}

// Plugin is the skeleton plugin implementation.
//
// Capability checklist (interfaces this struct implements):
//
//   - plugin.Plugin             — Name(), Description() (via BasePlugin)
//   - plugin.ConfigLoader       — DecodeAndSet, IsEnabled (via BasePlugin)
//   - plugin.CommandProvider    — Commands() in commands.go
//
// Add more capabilities (PipelineContributor, RuntimeProvider, etc.) by
// implementing their interfaces on this same struct — the framework
// discovers them via type assertion at runtime.
type Plugin struct {
	plugin.BasePlugin[*Config]
}

// Config is the typed plugin configuration decoded from
// `extensions.skeleton:` in `.terraci.yaml`.
type Config struct {
	// Enabled gates the plugin's behavior. Required because EnableMode is
	// EnabledExplicitly — the IsEnabledFn above reads this field.
	Enabled bool `yaml:"enabled" json:"enabled" jsonschema:"description=Enable the skeleton plugin,default=false"`

	// Greeting is a free-form example field demonstrating typed config.
	// jsonschema tags surface in `terraci schema` output for IDE auto-complete.
	Greeting string `yaml:"greeting,omitempty" json:"greeting,omitempty" jsonschema:"description=Greeting message printed by the command"`
}
