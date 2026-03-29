package config

import (
	"os"
	"path/filepath"
	"testing"

	"go.yaml.in/yaml/v4"
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
	return t.TempDir()
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	// Check structure defaults
	if cfg.Structure.Pattern != "{service}/{environment}/{region}/{module}" {
		t.Errorf("expected default pattern, got %q", cfg.Structure.Pattern)
	}
	if cfg.Plugins == nil {
		t.Error("expected non-nil Plugins map")
	}
}

func TestLoad(t *testing.T) {
	tmpDir := createTempDir(t)

	configContent := `
structure:
  pattern: "{service}/{env}/{region}/{module}"

plugins:
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

	// Verify gitlab config is in plugins map
	if _, ok := cfg.Plugins["gitlab"]; !ok {
		t.Fatal("expected gitlab in plugins map")
	}

	// Decode and check the config
	var glCfg map[string]any
	if err := cfg.PluginConfig("gitlab", &glCfg); err != nil {
		t.Fatal(err)
	}
	if glCfg["terraform_binary"] != "tofu" {
		t.Errorf("expected terraform_binary=tofu, got %v", glCfg["terraform_binary"])
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

plugins:
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

plugins:
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
			},
			wantErr: true,
			errMsg:  "structure.pattern is required",
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

func TestConfig_Clone(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Exclude = []string{"*/test/*"}
	cfg.Include = []string{"prod/*"}
	cfg.LibraryModules = &LibraryModulesConfig{Paths: []string{"_modules"}}

	var node yaml.Node
	if err := yaml.Unmarshal([]byte("enabled: true\nnested:\n  key: value\n"), &node); err != nil {
		t.Fatalf("yaml.Unmarshal() error = %v", err)
	}
	cfg.Plugins["summary"] = *node.Content[0]

	cloned := cfg.Clone()
	cloned.ServiceDir = "changed"
	cloned.Exclude[0] = "changed-exclude"
	cloned.Include[0] = "changed-include"
	cloned.LibraryModules.Paths[0] = "changed-path"

	pluginNode := cloned.Plugins["summary"]
	pluginNode.Content[0].Value = "disabled"
	cloned.Plugins["summary"] = pluginNode

	if cfg.ServiceDir != ".terraci" {
		t.Fatalf("original ServiceDir mutated: %q", cfg.ServiceDir)
	}
	if cfg.Exclude[0] != "*/test/*" {
		t.Fatalf("original Exclude mutated: %q", cfg.Exclude[0])
	}
	if cfg.Include[0] != "prod/*" {
		t.Fatalf("original Include mutated: %q", cfg.Include[0])
	}
	if cfg.LibraryModules.Paths[0] != "_modules" {
		t.Fatalf("original LibraryModules mutated: %q", cfg.LibraryModules.Paths[0])
	}
	if got := cfg.Plugins["summary"].Content[0].Value; got != "enabled" {
		t.Fatalf("original plugin yaml mutated: %q", got)
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

func TestPluginConfig(t *testing.T) {
	t.Run("nil plugins map returns nil error", func(t *testing.T) {
		cfg := &Config{}
		var target map[string]any
		if err := cfg.PluginConfig("missing", &target); err != nil {
			t.Errorf("expected nil error, got %v", err)
		}
	})

	t.Run("missing key returns nil error", func(t *testing.T) {
		cfg := DefaultConfig()
		var target map[string]any
		if err := cfg.PluginConfig("missing", &target); err != nil {
			t.Errorf("expected nil error, got %v", err)
		}
	})
}
