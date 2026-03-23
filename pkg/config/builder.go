package config

// Default images and runners for config initialization.
const (
	DefaultTerraformImage = "hashicorp/terraform:1.6"
	DefaultTofuImage      = "ghcr.io/opentofu/opentofu:1.6"
	DefaultGitHubRunner   = "ubuntu-latest"
	TerraCIImage          = "ghcr.io/edelwud/terraci:latest"
	binaryTofu            = "tofu"
)

// InitOptions holds all init parameters — shared between CLI and TUI.
type InitOptions struct {
	Provider    string
	Binary      string
	Pattern     string
	Image       string
	RunsOn      string
	PlanEnabled bool
	AutoApprove bool
	EnableMR    bool
	EnableCost  bool
}

// BuildConfig assembles a Config from the init options.
func (o *InitOptions) BuildConfig() *Config {
	newCfg := DefaultConfig()

	if o.Provider != "" {
		newCfg.Provider = o.Provider
	}
	if o.Pattern != "" {
		newCfg.Structure.Pattern = o.Pattern
	}
	switch o.ResolveProvider() {
	case ProviderGitHub:
		newCfg.GitHub = o.BuildGitHubConfig()
		newCfg.GitLab = nil
	default:
		newCfg.GitLab = o.BuildGitLabConfig()
		newCfg.GitHub = nil
	}

	if o.EnableCost {
		newCfg.Cost = &CostConfig{
			Enabled:       true,
			ShowInComment: true,
		}
	}

	return newCfg
}

// ResolveProvider returns the provider, defaulting to GitLab.
func (o *InitOptions) ResolveProvider() string {
	if o.Provider != "" {
		return o.Provider
	}
	return ProviderGitLab
}

// ResolveBinary returns the binary, defaulting to "terraform".
func (o *InitOptions) ResolveBinary() string {
	if o.Binary != "" {
		return o.Binary
	}
	return "terraform"
}

// ResolveImage returns the image, defaulting based on binary.
func (o *InitOptions) ResolveImage() string {
	if o.Image != "" {
		return o.Image
	}
	if o.ResolveBinary() == binaryTofu {
		return DefaultTofuImage
	}
	return DefaultTerraformImage
}

// BuildGitHubConfig creates a GitHubConfig from the init options.
func (o *InitOptions) BuildGitHubConfig() *GitHubConfig {
	ghCfg := &GitHubConfig{
		TerraformBinary: o.ResolveBinary(),
		RunsOn:          o.ResolveRunsOn(),
		PlanEnabled:     o.PlanEnabled,
		AutoApprove:     o.AutoApprove,
		InitEnabled:     true,
	}

	ghCfg.JobDefaults = &GitHubJobDefaults{
		StepsBefore: o.SetupStepsGitHub(),
	}

	if o.EnableMR {
		ghCfg.Permissions = map[string]string{
			"contents":      "read",
			"pull-requests": "write",
		}
		ghCfg.PR = &PRConfig{
			Comment: &MRCommentConfig{},
		}
	}

	return ghCfg
}

// ResolveRunsOn returns the GitHub runner, defaulting to "ubuntu-latest".
func (o *InitOptions) ResolveRunsOn() string {
	if o.RunsOn != "" {
		return o.RunsOn
	}
	return DefaultGitHubRunner
}

// SetupStepsGitHub returns the default setup steps for GitHub Actions.
func (o *InitOptions) SetupStepsGitHub() []GitHubStep {
	setupAction := "hashicorp/setup-terraform@v3"
	if o.ResolveBinary() == binaryTofu {
		setupAction = "opentofu/setup-opentofu@v1"
	}
	return []GitHubStep{
		{Uses: "actions/checkout@v4"},
		{Uses: setupAction},
	}
}

// BuildGitLabConfig creates a GitLabConfig from the init options.
func (o *InitOptions) BuildGitLabConfig() *GitLabConfig {
	glCfg := &GitLabConfig{
		TerraformBinary: o.ResolveBinary(),
		Image:           Image{Name: o.ResolveImage()},
		PlanEnabled:     o.PlanEnabled,
		AutoApprove:     o.AutoApprove,
		InitEnabled:     true,
	}

	if o.EnableMR {
		enabled := true
		glCfg.MR = &MRConfig{
			Comment: &MRCommentConfig{Enabled: &enabled},
			SummaryJob: &SummaryJobConfig{
				Image: &Image{Name: TerraCIImage},
			},
		}
	}

	return glCfg
}

// ApplyPlanOnly sets the PlanOnly and PlanEnabled flags for the given provider.
func ApplyPlanOnly(cfg *Config, provider string) {
	switch provider {
	case ProviderGitHub:
		if cfg.GitHub != nil {
			cfg.GitHub.PlanOnly = true
			cfg.GitHub.PlanEnabled = true
		}
	default:
		if cfg.GitLab != nil {
			cfg.GitLab.PlanOnly = true
			cfg.GitLab.PlanEnabled = true
		}
	}
}

// SetAutoApprove sets the AutoApprove flag for the given provider.
func SetAutoApprove(cfg *Config, provider string, approve bool) {
	switch provider {
	case ProviderGitHub:
		if cfg.GitHub != nil {
			cfg.GitHub.AutoApprove = approve
		}
	default:
		if cfg.GitLab != nil {
			cfg.GitLab.AutoApprove = approve
		}
	}
}
