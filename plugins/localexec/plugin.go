package localexec

import (
	"github.com/edelwud/terraci/pkg/plugin"
	pluginregistry "github.com/edelwud/terraci/pkg/plugin/registry"
)

func init() {
	pluginregistry.RegisterFactory(func() plugin.Plugin { return &Plugin{} })
}

const pluginName = "local-exec"

// Plugin runs the shared execution plan locally.
type Plugin struct{}

func (p *Plugin) Name() string        { return pluginName }
func (p *Plugin) Description() string { return "Execute terraci plans locally via terraform-exec" }
