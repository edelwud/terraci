// Package github provides the GitHub Actions plugin for TerraCi.
// It registers a pipeline generator and PR comment service.
package github

import (
	"context"
	"fmt"
	"os"

	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/graph"
	"github.com/edelwud/terraci/pkg/log"
	"github.com/edelwud/terraci/pkg/pipeline"
	"github.com/edelwud/terraci/pkg/plugin"
	githubci "github.com/edelwud/terraci/plugins/github/internal"
	"github.com/edelwud/terraci/plugins/policy"
)

func init() { //nolint:gochecknoinits // intentional plugin registration
	plugin.Register(&Plugin{})
}

const pluginName = "github"

// Re-export types from internal package for external consumers.
type (
	Config           = githubci.Config
	Image            = githubci.Image
	PRConfig         = githubci.PRConfig
	MRCommentConfig  = githubci.MRCommentConfig
	SummaryJobConfig = githubci.SummaryJobConfig
	PRContext        = githubci.PRContext
	Workflow         = githubci.Workflow
	WorkflowTrigger  = githubci.WorkflowTrigger
	PushTrigger      = githubci.PushTrigger
	PRTrigger        = githubci.PRTrigger
	Concurrency      = githubci.Concurrency
	Job              = githubci.Job
	Container        = githubci.Container
	Step             = githubci.Step
	Generator        = githubci.Generator
	Client           = githubci.Client
	PRService        = githubci.PRService
	JobDefaults      = githubci.JobDefaults
	JobOverwrite     = githubci.JobOverwrite
	JobOverwriteType = githubci.JobOverwriteType
	ConfigStep       = githubci.ConfigStep
)

// Re-export constants from internal package.
var (
	SummaryJobName     = githubci.SummaryJobName
	PolicyCheckJobName = githubci.PolicyCheckJobName
	OverwriteTypePlan  = githubci.OverwriteTypePlan
	OverwriteTypeApply = githubci.OverwriteTypeApply
)

// Re-export functions from internal package.
var (
	NewGenerator        = githubci.NewGenerator
	NewClient           = githubci.NewClient
	NewClientFromEnv    = githubci.NewClientFromEnv
	DetectPRContext     = githubci.DetectPRContext
	NewPRService        = githubci.NewPRService
	NewPRServiceFromEnv = githubci.NewPRServiceFromEnv
)

// Plugin is the GitHub Actions plugin.
type Plugin struct {
	cfg        *Config
	prCtx      *PRContext
	inCI       bool
	configured bool
}

func (p *Plugin) Name() string        { return pluginName }
func (p *Plugin) Description() string { return "GitHub Actions pipeline generation and PR comments" }

// ConfigProvider

func (p *Plugin) ConfigKey() string { return pluginName }
func (p *Plugin) NewConfig() any {
	return &Config{
		TerraformBinary: "terraform",
		RunsOn:          "ubuntu-latest",
		PlanEnabled:     true,
		InitEnabled:     true,
	}
}
func (p *Plugin) SetConfig(cfg any) error {
	gc, ok := cfg.(*Config)
	if !ok {
		return fmt.Errorf("expected *Config, got %T", cfg)
	}
	p.cfg = gc
	p.configured = true
	return nil
}

func (p *Plugin) IsConfigured() bool { return p.configured }

// Initializable — detect PR context at startup

func (p *Plugin) Initialize(_ context.Context, _ *plugin.AppContext) error {
	p.inCI = p.DetectEnv()
	if !p.inCI {
		return nil
	}

	p.prCtx = DetectPRContext()
	if p.prCtx.InPR {
		log.WithField("pr", p.prCtx.PRNumber).Debug("github: PR context detected")
	} else {
		log.Debug("github: Actions detected but not in PR workflow")
	}

	return nil
}

// GeneratorProvider

func (p *Plugin) ProviderName() string { return pluginName }
func (p *Plugin) DetectEnv() bool {
	return os.Getenv("GITHUB_ACTIONS") != ""
}

func (p *Plugin) NewGenerator(ctx *plugin.AppContext, depGraph *graph.DependencyGraph, modules []*discovery.Module) pipeline.Generator {
	cfg := p.cfg
	policyCfg := decodePolicyConfig(ctx)
	return githubci.NewGenerator(cfg, policyCfg, depGraph, modules)
}

func (p *Plugin) NewCommentService(_ *plugin.AppContext) ci.CommentService {
	cfg := p.cfg
	var prCfg *PRConfig
	if cfg != nil {
		prCfg = cfg.PR
	}
	return githubci.NewPRServiceFromEnv(prCfg)
}

// InitContributor — contributes GitHub Actions fields to the init wizard.

const defaultGitHubRunner = "ubuntu-latest"

func (p *Plugin) InitGroup() *plugin.InitGroupSpec {
	return &plugin.InitGroupSpec{
		Title: "GitHub Actions",
		Order: 100,
		Fields: []plugin.InitField{
			{
				Key:         "github.runs_on",
				Title:       "Runner Label",
				Description: "GitHub Actions runs-on value",
				Type:        "string",
				Default:     defaultGitHubRunner,
				Placeholder: defaultGitHubRunner,
			},
		},
		ShowWhen: func(s plugin.InitState) bool {
			return s.Provider() == "github"
		},
	}
}

func (p *Plugin) BuildInitConfig(state plugin.InitState) *plugin.InitContribution {
	if state.Provider() != "github" {
		return nil
	}
	binary := state.Binary()
	if binary == "" {
		binary = "terraform"
	}

	runsOn, _ := state.Get("github.runs_on").(string) //nolint:errcheck // safe type assertion
	if runsOn == "" {
		runsOn = defaultGitHubRunner
	}

	planEnabled, _ := state.Get("plan_enabled").(bool) //nolint:errcheck // safe type assertion
	autoApprove, _ := state.Get("auto_approve").(bool) //nolint:errcheck // safe type assertion
	enableMR, _ := state.Get("enable_mr").(bool)       //nolint:errcheck // safe type assertion

	setupAction := "hashicorp/setup-terraform@v3"
	if binary == "tofu" {
		setupAction = "opentofu/setup-opentofu@v1"
	}

	m := map[string]any{
		"terraform_binary": binary,
		"runs_on":          runsOn,
		"plan_enabled":     planEnabled,
		"auto_approve":     autoApprove,
		"init_enabled":     true,
		"job_defaults": map[string]any{
			"steps_before": []map[string]any{
				{"uses": "actions/checkout@v4"},
				{"uses": setupAction},
			},
		},
	}

	if enableMR {
		m["permissions"] = map[string]string{
			"contents":      "read",
			"pull-requests": "write",
		}
		m["pr"] = map[string]any{
			"comment": map[string]any{},
		}
	}

	return &plugin.InitContribution{PluginKey: "github", Config: m}
}

// decodePolicyConfig decodes the policy plugin config from the plugins map.
func decodePolicyConfig(ctx *plugin.AppContext) *policy.Config {
	var pc policy.Config
	if err := ctx.Config.PluginConfig("policy", &pc); err != nil {
		return nil
	}
	return &pc
}
