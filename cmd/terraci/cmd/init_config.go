package cmd

import "github.com/edelwud/terraci/pkg/config"

// Default images and runners for each binary.
const (
	defaultTerraformImage = "hashicorp/terraform:1.6"
	defaultTofuImage      = "ghcr.io/opentofu/opentofu:1.6"
	defaultGitHubRunner   = "ubuntu-latest"
	terraCIImage          = "ghcr.io/edelwud/terraci:latest"
)

// buildConfigFromFlags builds config from CLI flags (non-interactive mode).
func buildConfigFromFlags() *config.Config {
	opts := initOptions{
		provider: initProvider,
		binary:   initBinary,
		pattern:  initPattern,
		image:    initImage,
	}
	return opts.buildConfig()
}

// initOptions holds all init parameters — shared between CLI and TUI.
type initOptions struct {
	provider    string
	binary      string
	pattern     string
	image       string
	runsOn      string
	planEnabled bool
	autoApprove bool
	submodules  bool
	enableMR    bool
	enableCost  bool
}

func (o *initOptions) buildConfig() *config.Config {
	newCfg := config.DefaultConfig()

	if o.provider != "" {
		newCfg.Provider = o.provider
	}
	if o.pattern != "" {
		newCfg.Structure.Pattern = o.pattern
	}
	newCfg.Structure.AllowSubmodules = o.submodules
	if o.submodules {
		newCfg.Structure.MaxDepth = newCfg.Structure.MinDepth + 1
	}

	switch o.resolveProvider() {
	case config.ProviderGitHub:
		newCfg.GitHub = o.buildGitHubConfig()
		newCfg.GitLab = nil
	default:
		newCfg.GitLab = o.buildGitLabConfig()
		newCfg.GitHub = nil
	}

	if o.enableCost {
		newCfg.Cost = &config.CostConfig{
			Enabled:       true,
			ShowInComment: true,
		}
	}

	return newCfg
}

func (o *initOptions) resolveProvider() string {
	if o.provider != "" {
		return o.provider
	}
	return config.ProviderGitLab
}

func (o *initOptions) resolveBinary() string {
	if o.binary != "" {
		return o.binary
	}
	return "terraform"
}

func (o *initOptions) resolveImage() string {
	if o.image != "" {
		return o.image
	}
	if o.resolveBinary() == "tofu" {
		return defaultTofuImage
	}
	return defaultTerraformImage
}

// --- GitHub config ---

func (o *initOptions) buildGitHubConfig() *config.GitHubConfig {
	ghCfg := &config.GitHubConfig{
		TerraformBinary: o.resolveBinary(),
		RunsOn:          o.resolveRunsOn(),
		PlanEnabled:     o.planEnabled,
		AutoApprove:     o.autoApprove,
		InitEnabled:     true,
	}

	ghCfg.JobDefaults = &config.GitHubJobDefaults{
		StepsBefore: o.setupStepsGitHub(),
	}

	if o.enableMR {
		ghCfg.Permissions = map[string]string{
			"contents":      "read",
			"pull-requests": "write",
		}
		ghCfg.PR = &config.PRConfig{
			Comment: &config.MRCommentConfig{},
		}
	}

	return ghCfg
}

func (o *initOptions) resolveRunsOn() string {
	if o.runsOn != "" {
		return o.runsOn
	}
	return defaultGitHubRunner
}

func (o *initOptions) setupStepsGitHub() []config.GitHubStep {
	setupAction := "hashicorp/setup-terraform@v3"
	if o.resolveBinary() == "tofu" {
		setupAction = "opentofu/setup-opentofu@v1"
	}
	return []config.GitHubStep{
		{Uses: "actions/checkout@v4"},
		{Uses: setupAction},
	}
}

// --- GitLab config ---

func (o *initOptions) buildGitLabConfig() *config.GitLabConfig {
	glCfg := &config.GitLabConfig{
		TerraformBinary: o.resolveBinary(),
		Image:           config.Image{Name: o.resolveImage()},
		PlanEnabled:     o.planEnabled,
		AutoApprove:     o.autoApprove,
		InitEnabled:     true,
	}

	if o.enableMR {
		enabled := true
		glCfg.MR = &config.MRConfig{
			Comment: &config.MRCommentConfig{Enabled: &enabled},
			SummaryJob: &config.SummaryJobConfig{
				Image: &config.Image{Name: terraCIImage},
			},
		}
	}

	return glCfg
}
