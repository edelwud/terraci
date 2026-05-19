package plugin

import "github.com/spf13/cobra"

// CommandProvider adds CLI subcommands to TerraCi. The framework calls
// Commands() once during root command registration. Inside RunE, plugins
// should use CommandPlugin[T] to resolve the per-run AppContext plus the
// command-scoped plugin instance in one typed call.
type CommandProvider interface {
	Plugin
	Commands() []*cobra.Command
}

// VersionProvider plugins contribute version info to `terraci version`.
type VersionProvider interface {
	Plugin
	VersionInfo() map[string]string
}
