package gitlab

import (
	"os"

	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/graph"
	"github.com/edelwud/terraci/pkg/pipeline"
	"github.com/edelwud/terraci/pkg/plugin"
	gitlabci "github.com/edelwud/terraci/plugins/gitlab/internal"
)

// ProviderName returns the provider name.
func (p *Plugin) ProviderName() string { return pluginName }

// DetectEnv returns true if running in GitLab CI.
func (p *Plugin) DetectEnv() bool {
	return os.Getenv("GITLAB_CI") != "" || os.Getenv("CI_SERVER_URL") != ""
}

// NewGenerator creates a new GitLab CI pipeline generator.
func (p *Plugin) NewGenerator(_ *plugin.AppContext, depGraph *graph.DependencyGraph, modules []*discovery.Module) pipeline.Generator {
	contributions := plugin.CollectContributions()
	return gitlabci.NewGenerator(p.cfg, contributions, depGraph, modules)
}

// NewCommentService creates a new MR comment service.
func (p *Plugin) NewCommentService(_ *plugin.AppContext) ci.CommentService {
	var mrCfg *gitlabci.MRConfig
	if p.cfg != nil {
		mrCfg = p.cfg.MR
	}
	return gitlabci.NewMRServiceFromEnv(mrCfg)
}
