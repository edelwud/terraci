package config

import "testing"

func TestInitOptions_ResolveProvider(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		want     string
	}{
		{"default", "", ProviderGitLab},
		{"explicit gitlab", "gitlab", "gitlab"},
		{"explicit github", "github", "github"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &InitOptions{Provider: tt.provider}
			if got := o.ResolveProvider(); got != tt.want {
				t.Errorf("ResolveProvider() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestInitOptions_ResolveBinary(t *testing.T) {
	tests := []struct {
		name   string
		binary string
		want   string
	}{
		{"default", "", "terraform"},
		{"tofu", "tofu", "tofu"},
		{"custom", "terragrunt", "terragrunt"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &InitOptions{Binary: tt.binary}
			if got := o.ResolveBinary(); got != tt.want {
				t.Errorf("ResolveBinary() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestInitOptions_ResolveImage(t *testing.T) {
	tests := []struct {
		name   string
		binary string
		image  string
		want   string
	}{
		{"default terraform", "", "", DefaultTerraformImage},
		{"tofu default", "tofu", "", DefaultTofuImage},
		{"custom image", "", "my-image:v1", "my-image:v1"},
		{"custom overrides tofu", "tofu", "my-image:v1", "my-image:v1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &InitOptions{Binary: tt.binary, Image: tt.image}
			if got := o.ResolveImage(); got != tt.want {
				t.Errorf("ResolveImage() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestInitOptions_ResolveRunsOn(t *testing.T) {
	tests := []struct {
		name   string
		runsOn string
		want   string
	}{
		{"default", "", DefaultGitHubRunner},
		{"custom", "self-hosted", "self-hosted"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &InitOptions{RunsOn: tt.runsOn}
			if got := o.ResolveRunsOn(); got != tt.want {
				t.Errorf("ResolveRunsOn() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestInitOptions_SetupStepsGitHub(t *testing.T) {
	t.Run("terraform", func(t *testing.T) {
		o := &InitOptions{}
		steps := o.SetupStepsGitHub()
		if len(steps) != 2 {
			t.Fatalf("steps = %d, want 2", len(steps))
		}
		if steps[0].Uses != "actions/checkout@v4" {
			t.Errorf("step[0] = %q, want checkout", steps[0].Uses)
		}
		if steps[1].Uses != "hashicorp/setup-terraform@v3" {
			t.Errorf("step[1] = %q, want setup-terraform", steps[1].Uses)
		}
	})

	t.Run("tofu", func(t *testing.T) {
		o := &InitOptions{Binary: "tofu"}
		steps := o.SetupStepsGitHub()
		if steps[1].Uses != "opentofu/setup-opentofu@v1" {
			t.Errorf("step[1] = %q, want setup-opentofu", steps[1].Uses)
		}
	})
}

func TestInitOptions_BuildConfig_GitLab(t *testing.T) {
	o := &InitOptions{
		Pattern:     "{service}/{environment}/{module}",
		PlanEnabled: true,
		EnableMR:    true,
		EnableCost:  true,
	}

	cfg := o.BuildConfig()

	if cfg.GitLab == nil {
		t.Fatal("expected GitLab config")
	}
	if cfg.GitHub != nil {
		t.Error("expected GitHub to be nil for gitlab provider")
	}
	if cfg.Structure.Pattern != "{service}/{environment}/{module}" {
		t.Errorf("pattern = %q", cfg.Structure.Pattern)
	}
	if !cfg.GitLab.PlanEnabled {
		t.Error("PlanEnabled should be true")
	}
	if cfg.GitLab.MR == nil {
		t.Error("MR config should be set when EnableMR=true")
	}
	if cfg.Cost == nil || !cfg.Cost.Enabled {
		t.Error("Cost should be enabled")
	}
}

func TestInitOptions_BuildConfig_GitHub(t *testing.T) {
	o := &InitOptions{
		Provider:    "github",
		Binary:      "tofu",
		PlanEnabled: true,
		AutoApprove: true,
		EnableMR:    true,
	}

	cfg := o.BuildConfig()

	if cfg.GitHub == nil {
		t.Fatal("expected GitHub config")
	}
	if cfg.GitLab != nil {
		t.Error("expected GitLab to be nil for github provider")
	}
	if cfg.GitHub.TerraformBinary != "tofu" {
		t.Errorf("binary = %q, want tofu", cfg.GitHub.TerraformBinary)
	}
	if !cfg.GitHub.AutoApprove {
		t.Error("AutoApprove should be true")
	}
	if cfg.GitHub.PR == nil {
		t.Error("PR config should be set when EnableMR=true")
	}
	if cfg.GitHub.Permissions["pull-requests"] != "write" {
		t.Error("permissions should include pull-requests: write")
	}
}

func TestInitOptions_BuildGitLabConfig(t *testing.T) {
	t.Run("with MR enabled", func(t *testing.T) {
		o := &InitOptions{EnableMR: true}
		cfg := o.BuildGitLabConfig()
		if cfg.MR == nil {
			t.Fatal("MR should be set")
		}
		if cfg.MR.SummaryJob == nil || cfg.MR.SummaryJob.Image.Name != TerraCIImage {
			t.Error("summary job should use terraci image")
		}
	})

	t.Run("without MR", func(t *testing.T) {
		o := &InitOptions{}
		cfg := o.BuildGitLabConfig()
		if cfg.MR != nil {
			t.Error("MR should be nil when not enabled")
		}
	})
}

func TestApplyPlanOnly(t *testing.T) {
	t.Run("gitlab", func(t *testing.T) {
		cfg := &Config{GitLab: &GitLabConfig{}}
		ApplyPlanOnly(cfg, ProviderGitLab)
		if !cfg.GitLab.PlanOnly || !cfg.GitLab.PlanEnabled {
			t.Error("expected PlanOnly and PlanEnabled true")
		}
	})

	t.Run("github", func(t *testing.T) {
		cfg := &Config{GitHub: &GitHubConfig{}}
		ApplyPlanOnly(cfg, ProviderGitHub)
		if !cfg.GitHub.PlanOnly || !cfg.GitHub.PlanEnabled {
			t.Error("expected PlanOnly and PlanEnabled true")
		}
	})

	t.Run("nil gitlab config", func(_ *testing.T) {
		cfg := &Config{}
		ApplyPlanOnly(cfg, ProviderGitLab) // should not panic
	})

	t.Run("nil github config", func(_ *testing.T) {
		cfg := &Config{}
		ApplyPlanOnly(cfg, ProviderGitHub) // should not panic
	})
}

func TestSetAutoApprove(t *testing.T) {
	t.Run("gitlab enable", func(t *testing.T) {
		cfg := &Config{GitLab: &GitLabConfig{}}
		SetAutoApprove(cfg, ProviderGitLab, true)
		if !cfg.GitLab.AutoApprove {
			t.Error("expected AutoApprove true")
		}
	})

	t.Run("github disable", func(t *testing.T) {
		cfg := &Config{GitHub: &GitHubConfig{AutoApprove: true}}
		SetAutoApprove(cfg, ProviderGitHub, false)
		if cfg.GitHub.AutoApprove {
			t.Error("expected AutoApprove false")
		}
	})

	t.Run("nil configs", func(_ *testing.T) {
		cfg := &Config{}
		SetAutoApprove(cfg, ProviderGitLab, true) // should not panic
		SetAutoApprove(cfg, ProviderGitHub, true) // should not panic
	})
}
