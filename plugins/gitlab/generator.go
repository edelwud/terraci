package gitlab

import (
	"os"

	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/execution"
	"github.com/edelwud/terraci/pkg/graph"
	"github.com/edelwud/terraci/pkg/pipeline"
	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/pkg/plugin/registry"
	configpkg "github.com/edelwud/terraci/plugins/gitlab/internal/config"
	generatepkg "github.com/edelwud/terraci/plugins/gitlab/internal/generate"
	mrpkg "github.com/edelwud/terraci/plugins/gitlab/internal/mr"
)

// ProviderName returns the provider name.
func (p *Plugin) ProviderName() string { return p.Name() }

// DetectEnv returns true if running in GitLab CI.
func (p *Plugin) DetectEnv() bool {
	return os.Getenv("GITLAB_CI") != "" || os.Getenv("CI_SERVER_URL") != ""
}

// PipelineID returns the GitLab CI pipeline ID.
func (p *Plugin) PipelineID() string { return os.Getenv("CI_PIPELINE_ID") }

// CommitSHA returns the GitLab CI commit SHA.
func (p *Plugin) CommitSHA() string { return os.Getenv("CI_COMMIT_SHA") }

// NewGenerator creates a new GitLab CI pipeline generator.
func (p *Plugin) NewGenerator(ctx *plugin.AppContext, depGraph *graph.DependencyGraph, modules []*discovery.Module) pipeline.Generator {
	contributions := registry.CollectContributions(ctx)
	return generatepkg.NewGenerator(p.Config(), execution.ConfigFromProject(ctx.Config()), contributions, depGraph, modules)
}

// NewCommentService creates a new MR comment service.
func (p *Plugin) NewCommentService(_ *plugin.AppContext) ci.CommentService {
	var mrCfg *configpkg.MRConfig
	if p.Config() != nil {
		mrCfg = p.Config().MR
	}
	return mrpkg.NewServiceFromEnv(mrCfg)
}
