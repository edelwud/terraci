package config

import (
	"os"
	"path/filepath"
	"testing"
)

// writeTestConfig writes content to a config file
func writeTestConfig(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}
}

// createTempDir creates a temporary directory and returns cleanup function
func createTempDir(t *testing.T) string {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "config-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(tmpDir) })
	return tmpDir
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	// Check structure defaults
	if cfg.Structure.Pattern != "{service}/{environment}/{region}/{module}" {
		t.Errorf("expected default pattern, got %q", cfg.Structure.Pattern)
	}
	// Check GitLab defaults
	if cfg.GitLab.TerraformBinary != "terraform" {
		t.Errorf("expected TerraformBinary 'terraform', got %q", cfg.GitLab.TerraformBinary)
	}
	if cfg.GitLab.Image.Name != "hashicorp/terraform:1.6" {
		t.Errorf("expected Image 'hashicorp/terraform:1.6', got %q", cfg.GitLab.Image.Name)
	}
	if cfg.GitLab.StagesPrefix != "deploy" {
		t.Errorf("expected StagesPrefix 'deploy', got %q", cfg.GitLab.StagesPrefix)
	}
	if cfg.GitLab.Parallelism != 5 {
		t.Errorf("expected Parallelism 5, got %d", cfg.GitLab.Parallelism)
	}
	if !cfg.GitLab.PlanEnabled {
		t.Error("expected PlanEnabled to be true")
	}
	if cfg.GitLab.AutoApprove {
		t.Error("expected AutoApprove to be false")
	}
	if !cfg.GitLab.InitEnabled {
		t.Error("expected InitEnabled to be true")
	}
}

func TestLoad(t *testing.T) {
	tmpDir := createTempDir(t)

	configContent := `
structure:
  pattern: "{service}/{env}/{region}/{module}"

gitlab:
  image: hashicorp/terraform:1.7
  terraform_binary: tofu
  stages_prefix: infra
  parallelism: 10
  plan_enabled: false
  auto_approve: true

`
	configPath := filepath.Join(tmpDir, ".terraci.yaml")
	writeTestConfig(t, configPath, configContent)

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify loaded values
	if cfg.Structure.Pattern != "{service}/{env}/{region}/{module}" {
		t.Errorf("expected pattern, got %q", cfg.Structure.Pattern)
	}
	if cfg.GitLab.Image.Name != "hashicorp/terraform:1.7" {
		t.Errorf("expected Image name, got %q", cfg.GitLab.Image.Name)
	}
	if cfg.GitLab.TerraformBinary != "tofu" {
		t.Errorf("expected TerraformBinary 'tofu', got %q", cfg.GitLab.TerraformBinary)
	}
	if cfg.GitLab.StagesPrefix != "infra" {
		t.Errorf("expected StagesPrefix 'infra', got %q", cfg.GitLab.StagesPrefix)
	}
	if cfg.GitLab.Parallelism != 10 {
		t.Errorf("expected Parallelism 10, got %d", cfg.GitLab.Parallelism)
	}
	if cfg.GitLab.PlanEnabled {
		t.Error("expected PlanEnabled to be false")
	}
	if !cfg.GitLab.AutoApprove {
		t.Error("expected AutoApprove to be true")
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/path/.terraci.yaml")
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	tmpDir := createTempDir(t)

	invalidContent := `
structure:
  pattern: [invalid yaml
`
	configPath := filepath.Join(tmpDir, ".terraci.yaml")
	writeTestConfig(t, configPath, invalidContent)

	_, err := Load(configPath)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestLoadOrDefault(t *testing.T) {
	t.Run("loads config when file exists", func(t *testing.T) {
		tmpDir := createTempDir(t)

		configContent := `
structure:
  pattern: "{svc}/{env}/{region}/{mod}"

gitlab:
  image: custom/image:1.0
`
		configPath := filepath.Join(tmpDir, ".terraci.yaml")
		writeTestConfig(t, configPath, configContent)

		cfg, err := LoadOrDefault(tmpDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if cfg.Structure.Pattern != "{svc}/{env}/{region}/{mod}" {
			t.Errorf("expected loaded pattern, got %q", cfg.Structure.Pattern)
		}
	})

	t.Run("returns default when no config file", func(t *testing.T) {
		tmpDir := createTempDir(t)

		cfg, err := LoadOrDefault(tmpDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Should be default values
		if cfg.Structure.Pattern != "{service}/{environment}/{region}/{module}" {
			t.Errorf("expected default pattern, got %q", cfg.Structure.Pattern)
		}
	})

	t.Run("tries multiple config file names", func(t *testing.T) {
		tmpDir := createTempDir(t)

		// Use .terraci.yml extension
		configContent := `
structure:
  pattern: "{a}/{b}/{c}/{d}"

gitlab:
  image: test:1.0
`
		configPath := filepath.Join(tmpDir, ".terraci.yml")
		writeTestConfig(t, configPath, configContent)

		cfg, err := LoadOrDefault(tmpDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if cfg.Structure.Pattern != "{a}/{b}/{c}/{d}" {
			t.Errorf("expected pattern from .terraci.yml, got %q", cfg.Structure.Pattern)
		}
	})
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid default config",
			cfg:     DefaultConfig(),
			wantErr: false,
		},
		{
			name: "missing pattern",
			cfg: &Config{
				Structure: StructureConfig{
					Pattern: "",
				},
				GitLab: &GitLabConfig{
					Image: Image{Name: "test:1.0"},
				},
			},
			wantErr: true,
			errMsg:  "structure.pattern is required",
		},
		{
			name: "missing image",
			cfg: &Config{
				Provider: ProviderGitLab,
				Structure: StructureConfig{
					Pattern: "{service}/{env}/{region}/{module}",
				},
				GitLab: &GitLabConfig{
					Image: Image{Name: ""},
				},
			},
			wantErr: true,
			errMsg:  "gitlab.image is required",
		},
		{
			name: "invalid overwrite type",
			cfg: &Config{
				Provider: ProviderGitLab,
				Structure: StructureConfig{
					Pattern: "{service}/{env}/{region}/{module}",
				},
				GitLab: &GitLabConfig{
					Image: Image{Name: "test:1.0"},
					Overwrites: []JobOverwrite{
						{Type: "invalid"},
					},
				},
			},
			wantErr: true,
			errMsg:  "gitlab.overwrites[0].type must be 'plan' or 'apply'",
		},
		{
			name: "valid overwrite types",
			cfg: &Config{
				Structure: StructureConfig{
					Pattern: "{service}/{env}/{region}/{module}",
				},
				GitLab: &GitLabConfig{
					Image: Image{Name: "test:1.0"},
					Overwrites: []JobOverwrite{
						{Type: OverwriteTypePlan},
						{Type: OverwriteTypeApply},
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
					return
				}
				if tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Errorf("expected error %q, got %q", tt.errMsg, err.Error())
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestGitLabConfig_GetImage(t *testing.T) {
	cfg := &GitLabConfig{
		Image: Image{Name: "terraform:1.0"},
	}

	img := cfg.GetImage()
	if img.Name != "terraform:1.0" {
		t.Errorf("expected 'terraform:1.0', got %q", img.Name)
	}
}

func TestImage_UnmarshalYAML(t *testing.T) {
	tmpDir := createTempDir(t)

	t.Run("string format", func(t *testing.T) {
		configContent := `
structure:
  pattern: "{a}/{b}/{c}/{d}"

gitlab:
  image: hashicorp/terraform:1.6
`
		configPath := filepath.Join(tmpDir, "string.yaml")
		writeTestConfig(t, configPath, configContent)

		cfg, err := Load(configPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if cfg.GitLab.Image.Name != "hashicorp/terraform:1.6" {
			t.Errorf("expected image name, got %q", cfg.GitLab.Image.Name)
		}
		if cfg.GitLab.Image.HasEntrypoint() {
			t.Error("expected no entrypoint")
		}
	})

	t.Run("object format with entrypoint", func(t *testing.T) {
		configContent := `
structure:
  pattern: "{a}/{b}/{c}/{d}"

gitlab:
  image:
    name: hashicorp/terraform:1.6
    entrypoint: ["/bin/sh", "-c"]
`
		configPath := filepath.Join(tmpDir, "object.yaml")
		writeTestConfig(t, configPath, configContent)

		cfg, err := Load(configPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if cfg.GitLab.Image.Name != "hashicorp/terraform:1.6" {
			t.Errorf("expected image name, got %q", cfg.GitLab.Image.Name)
		}
		if !cfg.GitLab.Image.HasEntrypoint() {
			t.Error("expected entrypoint")
		}
		if len(cfg.GitLab.Image.Entrypoint) != 2 {
			t.Errorf("expected 2 entrypoint elements, got %d", len(cfg.GitLab.Image.Entrypoint))
		}
	})
}

func TestVaultSecret_UnmarshalYAML(t *testing.T) {
	tmpDir := createTempDir(t)

	t.Run("string shorthand", func(t *testing.T) {
		configContent := `
structure:
  pattern: "{a}/{b}/{c}/{d}"

gitlab:
  image: test:1.0
  job_defaults:
    secrets:
      AWS_SECRET:
        vault: secret/data/aws/key@production
`
		configPath := filepath.Join(tmpDir, "shorthand.yaml")
		writeTestConfig(t, configPath, configContent)

		cfg, err := Load(configPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		secret, ok := cfg.GitLab.JobDefaults.Secrets["AWS_SECRET"]
		if !ok {
			t.Fatal("missing AWS_SECRET")
		}
		if secret.Vault.Shorthand != "secret/data/aws/key@production" {
			t.Errorf("expected shorthand, got %q", secret.Vault.Shorthand)
		}
	})

	t.Run("full object syntax", func(t *testing.T) {
		configContent := `
structure:
  pattern: "{a}/{b}/{c}/{d}"

gitlab:
  image: test:1.0
  job_defaults:
    secrets:
      DB_PASSWORD:
        vault:
          path: secrets/data/database
          field: password
          engine:
            name: kv-v2
            path: secret
`
		configPath := filepath.Join(tmpDir, "object.yaml")
		writeTestConfig(t, configPath, configContent)

		cfg, err := Load(configPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		secret, ok := cfg.GitLab.JobDefaults.Secrets["DB_PASSWORD"]
		if !ok {
			t.Fatal("missing DB_PASSWORD")
		}
		if secret.Vault.Path != "secrets/data/database" {
			t.Errorf("expected path, got %q", secret.Vault.Path)
		}
		if secret.Vault.Field != "password" {
			t.Errorf("expected field, got %q", secret.Vault.Field)
		}
		if secret.Vault.Engine == nil {
			t.Fatal("expected engine")
		}
		if secret.Vault.Engine.Name != "kv-v2" {
			t.Errorf("expected engine name, got %q", secret.Vault.Engine.Name)
		}
	})
}

func TestImage_String(t *testing.T) {
	img := Image{Name: "test:1.0"}
	if img.String() != "test:1.0" {
		t.Errorf("expected String() to return name, got %q", img.String())
	}
}

func TestImage_HasEntrypoint(t *testing.T) {
	t.Run("without entrypoint", func(t *testing.T) {
		img := Image{Name: "test:1.0"}
		if img.HasEntrypoint() {
			t.Error("expected HasEntrypoint() to return false")
		}
	})

	t.Run("with entrypoint", func(t *testing.T) {
		img := Image{Name: "test:1.0", Entrypoint: []string{"/bin/sh"}}
		if !img.HasEntrypoint() {
			t.Error("expected HasEntrypoint() to return true")
		}
	})
}

func TestJobDefaults_ImplementsJobConfig(t *testing.T) {
	var _ JobConfig = &JobDefaults{}

	jd := &JobDefaults{
		Image:        &Image{Name: "test:1.0"},
		IDTokens:     map[string]IDToken{"GITLAB_OIDC": {Aud: "https://example.com"}},
		Secrets:      map[string]Secret{"SECRET": {File: true}},
		BeforeScript: []string{"echo before"},
		AfterScript:  []string{"echo after"},
		Artifacts:    &ArtifactsConfig{Paths: []string{"*.txt"}},
		Tags:         []string{"docker"},
		Rules:        []Rule{{If: "$CI_COMMIT_BRANCH == 'main'"}},
		Variables:    map[string]string{"VAR": "value"},
	}

	if jd.GetImage().Name != "test:1.0" {
		t.Error("GetImage() failed")
	}
	if len(jd.GetIDTokens()) != 1 {
		t.Error("GetIDTokens() failed")
	}
	if len(jd.GetSecrets()) != 1 {
		t.Error("GetSecrets() failed")
	}
	if len(jd.GetBeforeScript()) != 1 {
		t.Error("GetBeforeScript() failed")
	}
	if len(jd.GetAfterScript()) != 1 {
		t.Error("GetAfterScript() failed")
	}
	if jd.GetArtifacts() == nil {
		t.Error("GetArtifacts() failed")
	}
	if len(jd.GetTags()) != 1 {
		t.Error("GetTags() failed")
	}
	if len(jd.GetRules()) != 1 {
		t.Error("GetRules() failed")
	}
	if len(jd.GetVariables()) != 1 {
		t.Error("GetVariables() failed")
	}
}

func TestJobOverwrite_ImplementsJobConfig(t *testing.T) {
	var _ JobConfig = &JobOverwrite{}

	jo := &JobOverwrite{
		Type:         OverwriteTypePlan,
		Image:        &Image{Name: "plan:1.0"},
		IDTokens:     map[string]IDToken{"AWS_OIDC": {Aud: "sts.amazonaws.com"}},
		Secrets:      map[string]Secret{"AWS_KEY": {}},
		BeforeScript: []string{"aws configure"},
		AfterScript:  []string{"cleanup"},
		Artifacts:    &ArtifactsConfig{ExpireIn: "1 week"},
		Tags:         []string{"aws"},
		Rules:        []Rule{{When: "manual"}},
		Variables:    map[string]string{"AWS_REGION": "us-east-1"},
	}

	if jo.GetImage().Name != "plan:1.0" {
		t.Error("GetImage() failed")
	}
	if len(jo.GetIDTokens()) != 1 {
		t.Error("GetIDTokens() failed")
	}
	if len(jo.GetSecrets()) != 1 {
		t.Error("GetSecrets() failed")
	}
	if len(jo.GetBeforeScript()) != 1 {
		t.Error("GetBeforeScript() failed")
	}
	if len(jo.GetAfterScript()) != 1 {
		t.Error("GetAfterScript() failed")
	}
	if jo.GetArtifacts() == nil {
		t.Error("GetArtifacts() failed")
	}
	if len(jo.GetTags()) != 1 {
		t.Error("GetTags() failed")
	}
	if len(jo.GetRules()) != 1 {
		t.Error("GetRules() failed")
	}
	if len(jo.GetVariables()) != 1 {
		t.Error("GetVariables() failed")
	}
}

func TestConfig_Save(t *testing.T) {
	tmpDir := createTempDir(t)

	cfg := DefaultConfig()
	cfg.Structure.Pattern = "{svc}/{env}/{region}/{mod}"

	savePath := filepath.Join(tmpDir, "saved.yaml")
	if err := cfg.Save(savePath); err != nil {
		t.Fatalf("failed to save: %v", err)
	}

	// Read back and verify
	content, err := os.ReadFile(savePath)
	if err != nil {
		t.Fatalf("failed to read saved file: %v", err)
	}

	// Should have schema header
	if string(content[:30]) != "# yaml-language-server: $schem" {
		t.Errorf("expected schema header, got %q", string(content[:30]))
	}

	// Should be loadable
	loaded, err := Load(savePath)
	if err != nil {
		t.Fatalf("failed to load saved config: %v", err)
	}

	if loaded.Structure.Pattern != "{svc}/{env}/{region}/{mod}" {
		t.Errorf("expected pattern to be preserved, got %q", loaded.Structure.Pattern)
	}
}

func TestParsePatternSegmentCount(t *testing.T) {
	tests := []struct {
		pattern string
		want    int
	}{
		{"{service}/{environment}/{region}/{module}", 4},
		{"{a}/{b}/{c}", 3},
		{"{a}", 1},
		{"{a}/{b}/{c}/{d}/{e}", 5},
	}

	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			segments, err := ParsePattern(tt.pattern)
			if err != nil {
				t.Fatalf("ParsePattern(%q) error: %v", tt.pattern, err)
			}
			if len(segments) != tt.want {
				t.Errorf("len(ParsePattern(%q)) = %d, want %d", tt.pattern, len(segments), tt.want)
			}
		})
	}
}

func TestResolvedProvider(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		envKey   string
		envVal   string
		want     string
	}{
		{
			name:     "explicit provider set",
			provider: "github",
			want:     "github",
		},
		{
			name:   "GITHUB_ACTIONS env var set",
			envKey: "GITHUB_ACTIONS",
			envVal: "true",
			want:   "github",
		},
		{
			name:   "GITLAB_CI env var set",
			envKey: "GITLAB_CI",
			envVal: "true",
			want:   "gitlab",
		},
		{
			name: "no env no explicit defaults to gitlab",
			want: "gitlab",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear relevant env vars to avoid interference
			t.Setenv("GITHUB_ACTIONS", "")
			t.Setenv("GITLAB_CI", "")
			t.Setenv("CI_SERVER_URL", "")

			if tt.envKey != "" {
				t.Setenv(tt.envKey, tt.envVal)
			}

			cfg := &Config{Provider: tt.provider}
			got := cfg.ResolvedProvider()
			if got != tt.want {
				t.Errorf("ResolvedProvider() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetEffectiveConfig(t *testing.T) {
	t.Run("nil policy config", func(t *testing.T) {
		var p *PolicyConfig
		got := p.GetEffectiveConfig("some/module/path")
		if got != nil {
			t.Errorf("expected nil, got %+v", got)
		}
	})

	t.Run("no overwrites returns base config", func(t *testing.T) {
		p := &PolicyConfig{
			Enabled:   true,
			OnFailure: PolicyActionBlock,
			OnWarning: PolicyActionWarn,
		}
		got := p.GetEffectiveConfig("platform/stage/eu-central-1/vpc")
		if got == nil {
			t.Fatal("expected non-nil result")
		}
		if got.OnFailure != PolicyActionBlock {
			t.Errorf("expected OnFailure=block, got %q", got.OnFailure)
		}
		if got.OnWarning != PolicyActionWarn {
			t.Errorf("expected OnWarning=warn, got %q", got.OnWarning)
		}
	})

	t.Run("matching overwrite modifies config", func(t *testing.T) {
		enabled := false
		p := &PolicyConfig{
			Enabled:   true,
			OnFailure: PolicyActionBlock,
			OnWarning: PolicyActionWarn,
			Overwrites: []PolicyOverwrite{
				{
					Match:     "*/sandbox/*",
					OnFailure: PolicyActionWarn,
					Enabled:   &enabled,
				},
			},
		}
		got := p.GetEffectiveConfig("platform/sandbox/eu-central-1")
		if got == nil {
			t.Fatal("expected non-nil result")
		}
		if got.OnFailure != PolicyActionWarn {
			t.Errorf("expected OnFailure=warn after overwrite, got %q", got.OnFailure)
		}
		if got.Enabled {
			t.Error("expected Enabled=false after overwrite")
		}
	})
}

func TestValidate_NilGitLab(t *testing.T) {
	cfg := &Config{
		Structure: StructureConfig{
			Pattern: "{service}/{env}/{region}/{module}",
		},
		GitLab: nil,
		GitHub: &GitHubConfig{
			RunsOn: "ubuntu-latest",
		},
		Provider: "github",
	}

	err := cfg.Validate()
	if err != nil {
		t.Errorf("expected no error with nil GitLab and GitHub set, got: %v", err)
	}
}

func TestValidate_GitHubConfig(t *testing.T) {
	t.Run("empty RunsOn gets default", func(t *testing.T) {
		cfg := &Config{
			Structure: StructureConfig{
				Pattern: "{service}/{env}/{region}/{module}",
			},
			Provider: "github",
			GitHub: &GitHubConfig{
				RunsOn: "",
			},
		}

		err := cfg.Validate()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if cfg.GitHub.RunsOn != "ubuntu-latest" {
			t.Errorf("expected RunsOn to be set to ubuntu-latest, got %q", cfg.GitHub.RunsOn)
		}
	})

	t.Run("explicit RunsOn preserved", func(t *testing.T) {
		cfg := &Config{
			Structure: StructureConfig{
				Pattern: "{service}/{env}/{region}/{module}",
			},
			Provider: "github",
			GitHub: &GitHubConfig{
				RunsOn: "self-hosted",
			},
		}

		err := cfg.Validate()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if cfg.GitHub.RunsOn != "self-hosted" {
			t.Errorf("expected RunsOn to remain self-hosted, got %q", cfg.GitHub.RunsOn)
		}
	})
}

func TestMatchGlob_DoubleStarPatterns(t *testing.T) {
	tests := []struct {
		pattern string
		path    string
		want    bool
	}{
		// Single * matches one segment
		{"*/stage/*/*", "platform/stage/eu-central-1/vpc", true},
		{"*/stage/*/*", "platform/prod/eu-central-1/vpc", false},
		{"*/stage/*/*", "platform/stage/eu-central-1/ec2/rabbitmq", false}, // too many segments

		// ** matches any number of segments
		{"**/sandbox/**", "platform/sandbox/eu-central-1/test", true},
		{"**/sandbox/**", "platform/stage/eu-central-1/test", false}, // no "sandbox"
		{"**/sandbox/**", "sandbox/test", true},                      // sandbox at start
		{"**/sandbox/**", "a/b/sandbox/c/d/e", true},                 // sandbox in middle

		// ** prefix
		{"legacy/**", "legacy/old/eu-central-1/db", true},
		{"legacy/**", "platform/legacy/module", false}, // legacy not at start

		// ** suffix
		{"**/vpc", "platform/stage/eu-central-1/vpc", true},
		{"**/vpc", "platform/stage/eu-central-1/eks", false},

		// ** middle
		{"platform/**/vpc", "platform/stage/eu-central-1/vpc", true},
		{"platform/**/vpc", "platform/vpc", true},                  // zero segments between
		{"platform/**/vpc", "other/stage/eu-central-1/vpc", false}, // wrong prefix

		// Exact match (no wildcards)
		{"platform/stage/eu-central-1/vpc", "platform/stage/eu-central-1/vpc", true},
		{"platform/stage/eu-central-1/vpc", "platform/stage/eu-central-1/eks", false},

		// The bug case: **/sandbox/** must NOT match paths without "sandbox"
		{"**/sandbox/**", "platform/stage/eu-central-1/bad", false},
		{"**/sandbox/**", "platform/stage/eu-central-1/app", false},
		{"**/sandbox/**", "legacy/old/eu-central-1/db", false},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"_vs_"+tt.path, func(t *testing.T) {
			got, err := matchGlob(tt.pattern, tt.path)
			if err != nil {
				t.Fatalf("matchGlob(%q, %q) error: %v", tt.pattern, tt.path, err)
			}
			if got != tt.want {
				t.Errorf("matchGlob(%q, %q) = %v, want %v", tt.pattern, tt.path, got, tt.want)
			}
		})
	}
}

func TestGetEffectiveConfig_Overwrites(t *testing.T) {
	enabled := true
	disabled := false

	base := &PolicyConfig{
		Enabled:    true,
		Namespaces: []string{"terraform"},
		OnFailure:  PolicyActionBlock,
		OnWarning:  PolicyActionWarn,
		Overwrites: []PolicyOverwrite{
			{Match: "**/sandbox/**", OnFailure: PolicyActionWarn},
			{Match: "legacy/**", Enabled: &disabled},
			{Match: "**/prod/**", Namespaces: []string{"terraform", "compliance"}, OnWarning: PolicyActionBlock},
		},
	}

	t.Run("nil policy", func(t *testing.T) {
		var p *PolicyConfig
		if got := p.GetEffectiveConfig("anything"); got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})

	t.Run("no matching overwrite returns base", func(t *testing.T) {
		eff := base.GetEffectiveConfig("platform/stage/eu-central-1/vpc")
		if eff.OnFailure != PolicyActionBlock {
			t.Errorf("OnFailure = %q, want block", eff.OnFailure)
		}
		if !eff.Enabled {
			t.Error("Enabled should be true")
		}
	})

	t.Run("sandbox overwrite changes on_failure to warn", func(t *testing.T) {
		eff := base.GetEffectiveConfig("platform/sandbox/eu-central-1/test")
		if eff.OnFailure != PolicyActionWarn {
			t.Errorf("OnFailure = %q, want warn", eff.OnFailure)
		}
		if !eff.Enabled {
			t.Error("sandbox should still be enabled")
		}
	})

	t.Run("legacy overwrite disables checks", func(t *testing.T) {
		eff := base.GetEffectiveConfig("legacy/old/eu-central-1/db")
		if eff.Enabled {
			t.Error("legacy should be disabled")
		}
	})

	t.Run("prod overwrite adds namespace and changes on_warning", func(t *testing.T) {
		eff := base.GetEffectiveConfig("platform/prod/eu-central-1/vpc")
		if len(eff.Namespaces) != 2 {
			t.Errorf("Namespaces = %v, want 2 entries", eff.Namespaces)
		}
		if eff.OnWarning != PolicyActionBlock {
			t.Errorf("OnWarning = %q, want block", eff.OnWarning)
		}
	})

	t.Run("re-enable overridden module", func(t *testing.T) {
		cfg := &PolicyConfig{
			Enabled:   true,
			OnFailure: PolicyActionBlock,
			Overwrites: []PolicyOverwrite{
				{Match: "legacy/**", Enabled: &disabled},
				{Match: "legacy/critical/**", Enabled: &enabled}, // re-enable critical legacy
			},
		}
		eff := cfg.GetEffectiveConfig("legacy/critical/eu-central-1/auth")
		if !eff.Enabled {
			t.Error("critical legacy should be re-enabled by second overwrite")
		}
	})
}
