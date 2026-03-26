// Package summary provides the summary plugin for TerraCi.
// It registers a pipeline contributor (PhaseFinalize) and the `terraci summary` command.
package summary

import "github.com/edelwud/terraci/pkg/plugin"

func init() { //nolint:gochecknoinits // intentional plugin registration
	plugin.Register(&Plugin{})
}

// Plugin is the summary plugin.
type Plugin struct {
	cfg        *Config
	configured bool
}

func (p *Plugin) Name() string        { return "summary" }
func (p *Plugin) Description() string { return "MR/PR comment posting from plan results" }
