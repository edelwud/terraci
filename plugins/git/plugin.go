// Package git provides the Git change detection plugin for TerraCi.
package git

import (
	"github.com/edelwud/terraci/pkg/plugin"
)

const pluginName = "git"

func init() {
	plugin.Register(&Plugin{})
}

// Plugin is the Git change detection plugin.
type Plugin struct{}

func (p *Plugin) Name() string        { return pluginName }
func (p *Plugin) Description() string { return "Git change detection for incremental pipelines" }
func (p *Plugin) Reset()              {}
