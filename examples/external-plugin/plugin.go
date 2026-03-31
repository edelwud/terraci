// Package hello is an example external plugin for TerraCi.
// It adds a `terraci hello` command that prints discovered Terraform modules.
//
// Build with xterraci:
//
//	xterraci build --with github.com/edelwud/terraci/examples/external-plugin=./examples/external-plugin
package hello

import (
	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/pkg/plugin/registry"
)

func init() {
	registry.Register(&Plugin{})
}

// Plugin is the example "hello" plugin.
type Plugin struct {
	plugin.BasePlugin[*Config]
}

// Config holds optional plugin configuration under plugins.hello in .terraci.yaml.
type Config struct {
	Greeting string `yaml:"greeting"`
}
