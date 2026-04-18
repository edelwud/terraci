package localexec

import pluginregistry "github.com/edelwud/terraci/pkg/plugin/registry"

func init() {
	pluginregistry.Register(&Plugin{})
}

// Plugin runs the shared execution plan locally.
type Plugin struct{}

func (p *Plugin) Name() string        { return "local-exec" }
func (p *Plugin) Description() string { return "Execute terraci plans locally via terraform-exec" }
