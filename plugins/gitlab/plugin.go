// Package gitlab provides the GitLab CI plugin for TerraCi.
// It registers a pipeline generator and MR comment service.
package gitlab

import (
	"github.com/edelwud/terraci/pkg/plugin"
	gitlabci "github.com/edelwud/terraci/plugins/gitlab/internal"
)

func init() { //nolint:gochecknoinits // intentional plugin registration
	plugin.Register(&Plugin{})
}

const pluginName = "gitlab"

// Plugin is the GitLab CI plugin.
type Plugin struct {
	cfg        *gitlabci.Config
	mrCtx      *gitlabci.MRContext
	inCI       bool
	configured bool
}

func (p *Plugin) Name() string        { return pluginName }
func (p *Plugin) Description() string { return "GitLab CI pipeline generation and MR comments" }
