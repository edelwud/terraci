// Package github provides the GitHub Actions plugin for TerraCi.
// It registers a pipeline generator and PR comment service.
package github

import (
	"github.com/edelwud/terraci/pkg/plugin"
	githubci "github.com/edelwud/terraci/plugins/github/internal"
)

func init() { //nolint:gochecknoinits // intentional plugin registration
	plugin.Register(&Plugin{})
}

const pluginName = "github"

// Plugin is the GitHub Actions plugin.
type Plugin struct {
	cfg        *githubci.Config
	prCtx      *githubci.PRContext
	inCI       bool
	configured bool
}

func (p *Plugin) Name() string        { return pluginName }
func (p *Plugin) Description() string { return "GitHub Actions pipeline generation and PR comments" }
