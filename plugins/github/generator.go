package github

import (
	"os"

	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/execution"
	"github.com/edelwud/terraci/pkg/pipeline"
	"github.com/edelwud/terraci/pkg/plugin"
	configpkg "github.com/edelwud/terraci/plugins/github/internal/config"
	generatepkg "github.com/edelwud/terraci/plugins/github/internal/generate"
	prpkg "github.com/edelwud/terraci/plugins/github/internal/pr"
)

// ProviderName returns the provider name.
func (p *Plugin) ProviderName() string { return p.Name() }

// DetectEnv returns true if running in GitHub Actions.
func (p *Plugin) DetectEnv() bool {
	return os.Getenv("GITHUB_ACTIONS") != ""
}

// PipelineID returns the GitHub Actions run ID.
func (p *Plugin) PipelineID() string { return os.Getenv("GITHUB_RUN_ID") }

// CommitSHA returns the GitHub Actions commit SHA.
func (p *Plugin) CommitSHA() string { return os.Getenv("GITHUB_SHA") }

// NewGenerator creates a new GitHub Actions pipeline generator bound to the
// pre-built IR.
func (p *Plugin) NewGenerator(ctx *plugin.AppContext, ir *pipeline.IR) pipeline.Generator {
	return generatepkg.NewGenerator(p.Config(), execution.ConfigFromProject(ctx.Config()), ir)
}

// NewCommentService creates a new PR comment service.
func (p *Plugin) NewCommentService(_ *plugin.AppContext) ci.CommentService {
	var prCfg *configpkg.PRConfig
	if p.Config() != nil {
		prCfg = p.Config().PR
	}
	return prpkg.NewServiceFromEnv(prCfg)
}
