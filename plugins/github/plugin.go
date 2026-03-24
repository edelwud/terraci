// Package github provides the GitHub Actions plugin for TerraCi.
// It registers a pipeline generator and PR comment service.
package github

import (
	"os"

	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/config"
	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/graph"
	"github.com/edelwud/terraci/pkg/pipeline"
	"github.com/edelwud/terraci/pkg/plugin"
)

func init() {
	plugin.Register(&Plugin{})
}

// Plugin is the GitHub Actions plugin.
type Plugin struct {
	cfg *config.GitHubConfig
}

func (p *Plugin) Name() string        { return "github" }
func (p *Plugin) Description() string { return "GitHub Actions pipeline generation and PR comments" }

// ConfigProvider

func (p *Plugin) ConfigKey() string { return "github" }
func (p *Plugin) NewConfig() any {
	return &config.GitHubConfig{
		TerraformBinary: "terraform",
		RunsOn:          "ubuntu-latest",
		PlanEnabled:     true,
		InitEnabled:     true,
	}
}
func (p *Plugin) SetConfig(cfg any) error {
	p.cfg = cfg.(*config.GitHubConfig)
	return nil
}

// GeneratorProvider

func (p *Plugin) ProviderName() string { return "github" }
func (p *Plugin) DetectEnv() bool {
	return os.Getenv("GITHUB_ACTIONS") != ""
}

func (p *Plugin) NewGenerator(ctx *plugin.AppContext, depGraph *graph.DependencyGraph, modules []*discovery.Module) pipeline.Generator {
	cfg := p.cfg
	if cfg == nil {
		cfg = ctx.Config.GitHub
	}
	return NewGenerator(cfg, ctx.Config.Policy, depGraph, modules)
}

func (p *Plugin) NewCommentService(ctx *plugin.AppContext) ci.CommentService {
	cfg := p.cfg
	if cfg == nil {
		cfg = ctx.Config.GitHub
	}
	var prCfg *config.PRConfig
	if cfg != nil {
		prCfg = cfg.PR
	}
	return NewPRServiceFromEnv(prCfg)
}
