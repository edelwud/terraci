// Package gitlab provides the GitLab CI plugin for TerraCi.
// It registers a pipeline generator and MR comment service.
package gitlab

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

// Plugin is the GitLab CI plugin.
type Plugin struct {
	cfg *config.GitLabConfig
}

func (p *Plugin) Name() string        { return "gitlab" }
func (p *Plugin) Description() string { return "GitLab CI pipeline generation and MR comments" }

// ConfigProvider

func (p *Plugin) ConfigKey() string { return "gitlab" }
func (p *Plugin) NewConfig() any {
	return &config.GitLabConfig{
		TerraformBinary: "terraform",
		Image:           config.Image{Name: "hashicorp/terraform:1.6"},
		StagesPrefix:    "deploy",
		Parallelism:     5,
		PlanEnabled:     true,
		InitEnabled:     true,
	}
}
func (p *Plugin) SetConfig(cfg any) error {
	p.cfg = cfg.(*config.GitLabConfig)
	return nil
}

// GeneratorProvider

func (p *Plugin) ProviderName() string { return "gitlab" }
func (p *Plugin) DetectEnv() bool {
	return os.Getenv("GITLAB_CI") != "" || os.Getenv("CI_SERVER_URL") != ""
}

func (p *Plugin) NewGenerator(ctx *plugin.AppContext, depGraph *graph.DependencyGraph, modules []*discovery.Module) pipeline.Generator {
	cfg := p.cfg
	if cfg == nil {
		cfg = ctx.Config.GitLab
	}
	return NewGenerator(cfg, ctx.Config.Policy, depGraph, modules)
}

func (p *Plugin) NewCommentService(ctx *plugin.AppContext) ci.CommentService {
	cfg := p.cfg
	if cfg == nil {
		cfg = ctx.Config.GitLab
	}
	var mrCfg *config.MRConfig
	if cfg != nil {
		mrCfg = cfg.MR
	}
	return NewMRServiceFromEnv(mrCfg)
}
