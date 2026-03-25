// Package gitlab provides the GitLab CI plugin for TerraCi.
// It registers a pipeline generator and MR comment service.
package gitlab

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
	gitlabci "github.com/edelwud/terraci/plugins/gitlab/internal"
)

func init() { //nolint:gochecknoinits // intentional plugin registration
	plugin.Register(&Plugin{})
}

const pluginName = "gitlab"

// Re-export types from internal package for external consumers.
type (
	Config           = gitlabci.Config
	Image            = gitlabci.Image
	MRConfig         = gitlabci.MRConfig
	MRCommentConfig  = gitlabci.MRCommentConfig
	SummaryJobConfig = gitlabci.SummaryJobConfig
	MRContext        = gitlabci.MRContext
	Pipeline         = gitlabci.Pipeline
	Job              = gitlabci.Job
	JobNeed          = gitlabci.JobNeed
	Rule             = gitlabci.Rule
	ImageConfig      = gitlabci.ImageConfig
	DefaultConfig    = gitlabci.DefaultConfig
	Workflow         = gitlabci.Workflow
	Artifacts        = gitlabci.Artifacts
	Reports          = gitlabci.Reports
	Cache            = gitlabci.Cache
	Secret           = gitlabci.Secret
	VaultSecret      = gitlabci.VaultSecret
	VaultEngine      = gitlabci.VaultEngine
	IDToken          = gitlabci.IDToken
	Generator        = gitlabci.Generator
	Client           = gitlabci.Client
	MRService        = gitlabci.MRService
	JobConfig        = gitlabci.JobConfig
	JobDefaults      = gitlabci.JobDefaults
	JobOverwrite     = gitlabci.JobOverwrite
	JobOverwriteType = gitlabci.JobOverwriteType
	ArtifactsConfig  = gitlabci.ArtifactsConfig
	ArtifactReports  = gitlabci.ArtifactReports
	CfgSecret        = gitlabci.CfgSecret
	CfgVaultSecret   = gitlabci.CfgVaultSecret
)

// Re-export constants from internal package.
var (
	DefaultStagesPrefix = gitlabci.DefaultStagesPrefix
	SummaryJobName      = gitlabci.SummaryJobName
	SummaryStageName    = gitlabci.SummaryStageName
	WhenManual          = gitlabci.WhenManual
	OverwriteTypePlan   = gitlabci.OverwriteTypePlan
	OverwriteTypeApply  = gitlabci.OverwriteTypeApply
)

// Re-export functions from internal package.
var (
	NewGenerator        = gitlabci.NewGenerator
	NewClient           = gitlabci.NewClient
	NewClientFromEnv    = gitlabci.NewClientFromEnv
	DetectMRContext     = gitlabci.DetectMRContext
	NewMRService        = gitlabci.NewMRService
	NewMRServiceFromEnv = gitlabci.NewMRServiceFromEnv
	FindTerraCIComment  = gitlabci.FindTerraCIComment
)

// Plugin is the GitLab CI plugin.
type Plugin struct {
	cfg        *Config
	mrCtx      *MRContext
	inCI       bool
	configured bool
}

func (p *Plugin) Name() string        { return pluginName }
func (p *Plugin) Description() string { return "GitLab CI pipeline generation and MR comments" }

// ConfigProvider

func (p *Plugin) ConfigKey() string { return pluginName }
func (p *Plugin) NewConfig() any {
	return &Config{
		TerraformBinary: "terraform",
		Image:           Image{Name: "hashicorp/terraform:1.6"},
		StagesPrefix:    "deploy",
		Parallelism:     5,
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

// Initializable — detect MR context at startup

func (p *Plugin) Initialize(_ context.Context, _ *plugin.AppContext) error {
	p.inCI = p.DetectEnv()
	if !p.inCI {
		return nil
	}

	p.mrCtx = DetectMRContext()
	if p.mrCtx.InMR {
		log.WithField("mr", p.mrCtx.MRIID).Debug("gitlab: MR context detected")
	} else {
		log.Debug("gitlab: CI detected but not in MR pipeline")
	}

	return nil
}

// GeneratorProvider

func (p *Plugin) ProviderName() string { return pluginName }
func (p *Plugin) DetectEnv() bool {
	return os.Getenv("GITLAB_CI") != "" || os.Getenv("CI_SERVER_URL") != ""
}

func (p *Plugin) NewGenerator(_ *plugin.AppContext, depGraph *graph.DependencyGraph, modules []*discovery.Module) pipeline.Generator {
	contributions := collectContributions()
	return gitlabci.NewGenerator(p.cfg, contributions, depGraph, modules)
}

func (p *Plugin) NewCommentService(_ *plugin.AppContext) ci.CommentService {
	cfg := p.cfg
	var mrCfg *MRConfig
	if cfg != nil {
		mrCfg = cfg.MR
	}
	return gitlabci.NewMRServiceFromEnv(mrCfg)
}

// InitContributor — contributes GitLab CI fields to the init wizard.

const (
	defaultTerraformImage = "hashicorp/terraform:1.6"
	defaultTofuImage      = "ghcr.io/opentofu/opentofu:1.6"
	terraCIImage          = "ghcr.io/edelwud/terraci:latest"
)

func (p *Plugin) InitGroup() *plugin.InitGroupSpec {
	return &plugin.InitGroupSpec{
		Title: "GitLab CI",
		Order: 100,
		Fields: []plugin.InitField{
			{
				Key:         "gitlab.image",
				Title:       "Docker Image",
				Description: "Base Docker image for terraform jobs",
				Type:        "string",
				Default:     defaultTerraformImage,
				Placeholder: defaultTerraformImage,
			},
		},
		ShowWhen: func(s plugin.InitState) bool {
			return s.Provider() == "gitlab"
		},
	}
}

func (p *Plugin) BuildInitConfig(state plugin.InitState) *plugin.InitContribution {
	if state.Provider() != "gitlab" {
		return nil
	}
	binary := state.Binary()
	if binary == "" {
		binary = "terraform"
	}

	image, _ := state.Get("gitlab.image").(string) //nolint:errcheck // safe type assertion
	if image == "" {
		if binary == "tofu" {
			image = defaultTofuImage
		} else {
			image = defaultTerraformImage
		}
	}

	planEnabled, _ := state.Get("plan_enabled").(bool) //nolint:errcheck // safe type assertion
	autoApprove, _ := state.Get("auto_approve").(bool) //nolint:errcheck // safe type assertion
	enableMR, _ := state.Get("enable_mr").(bool)       //nolint:errcheck // safe type assertion

	m := map[string]any{
		"terraform_binary": binary,
		"image":            map[string]any{"name": image},
		"plan_enabled":     planEnabled,
		"auto_approve":     autoApprove,
		"init_enabled":     true,
	}
	if enableMR {
		m["mr"] = map[string]any{
			"comment": map[string]any{"enabled": true},
			"summary_job": map[string]any{
				"image": map[string]any{"name": terraCIImage},
			},
		}
	}
	return &plugin.InitContribution{PluginKey: "gitlab", Config: m}
}

// collectContributions gathers pipeline contributions from all PipelineContributor plugins.
func collectContributions() []*pipeline.Contribution {
	contributors := plugin.ByCapability[plugin.PipelineContributor]()
	contributions := make([]*pipeline.Contribution, 0, len(contributors))
	for _, c := range contributors {
		contributions = append(contributions, c.PipelineContribution())
	}
	return contributions
}
