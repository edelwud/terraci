package github

import (
	"os"

	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/graph"
	"github.com/edelwud/terraci/pkg/pipeline"
	"github.com/edelwud/terraci/pkg/plugin"
	githubci "github.com/edelwud/terraci/plugins/github/internal"
)

// ProviderName returns the provider name.
func (p *Plugin) ProviderName() string { return pluginName }

// DetectEnv returns true if running in GitHub Actions.
func (p *Plugin) DetectEnv() bool {
	return os.Getenv("GITHUB_ACTIONS") != ""
}

// NewGenerator creates a new GitHub Actions pipeline generator.
func (p *Plugin) NewGenerator(_ *plugin.AppContext, depGraph *graph.DependencyGraph, modules []*discovery.Module) pipeline.Generator {
	contributions := plugin.CollectContributions()
	return githubci.NewGenerator(p.cfg, contributions, depGraph, modules)
}

// NewCommentService creates a new PR comment service.
func (p *Plugin) NewCommentService(_ *plugin.AppContext) ci.CommentService {
	cfg := p.cfg
	var prCfg *githubci.PRConfig
	if cfg != nil {
		prCfg = cfg.PR
	}
	return githubci.NewPRServiceFromEnv(prCfg)
}
