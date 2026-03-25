// Package git provides the Git change detection plugin for TerraCi.
package git

import (
	"github.com/edelwud/terraci/pkg/plugin"
	gitclient "github.com/edelwud/terraci/plugins/git/internal"
)

func init() { //nolint:gochecknoinits // intentional plugin registration
	plugin.Register(&Plugin{})
}

// Plugin is the Git change detection plugin.
type Plugin struct {
	client     *gitclient.Client
	defaultRef string
	isRepo     bool
}

func (p *Plugin) Name() string        { return "git" }
func (p *Plugin) Description() string { return "Git change detection for incremental pipelines" }
