// Package summary provides the summary plugin for TerraCi.
// It registers a pipeline contributor (PhaseFinalize) and the `terraci summary` command.
package summary

import (
	"github.com/edelwud/terraci/pkg/plugin"
	summaryengine "github.com/edelwud/terraci/plugins/summary/internal"
)

const pluginName = "summary"

func init() { //nolint:gochecknoinits // intentional plugin registration
	plugin.Register(&Plugin{})
}

// Plugin is the summary plugin.
type Plugin struct {
	cfg        *summaryengine.Config
	configured bool
}

func (p *Plugin) Name() string        { return pluginName }
func (p *Plugin) Description() string { return "MR/PR comment posting from plan results" }
func (p *Plugin) Reset()              { *p = Plugin{} }
