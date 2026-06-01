package cishell

import (
	"testing"

	"github.com/edelwud/terraci/pkg/pipeline"
)

func TestTerraformJobConfig_PlanScript(t *testing.T) {
	t.Parallel()

	modulePath := "svc/prod/us-east-1/vpc"

	tests := []struct {
		name              string
		config            pipeline.TerraformJobConfig
		outputs           pipeline.PlanOutputs
		wantInitCmd       bool
		wantDetailedCmds  bool
		wantSimplePlan    bool
		wantArtifactCount int
	}{
		{
			name:              "InitEnabled adds init command",
			config:            mustTerraformConfig(t, true, "terraform"),
			wantInitCmd:       true,
			wantSimplePlan:    true,
			wantArtifactCount: 1,
		},
		{
			name:              "InitEnabled false skips init",
			config:            mustTerraformConfig(t, false, "terraform"),
			wantInitCmd:       false,
			wantSimplePlan:    true,
			wantArtifactCount: 1,
		},
		{
			name:              "DetailedPlan adds tee show json commands",
			config:            mustTerraformConfig(t, false, "terraform"),
			outputs:           pipeline.PlanOutputs{Text: true, JSON: true},
			wantInitCmd:       false,
			wantDetailedCmds:  true,
			wantArtifactCount: 3,
		},
		{
			name:              "DetailedPlan with init",
			config:            mustTerraformConfig(t, true, "terraform"),
			outputs:           pipeline.PlanOutputs{Text: true, JSON: true},
			wantInitCmd:       true,
			wantDetailedCmds:  true,
			wantArtifactCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			op, _, artifact := tt.config.NewPlanOperation("plan-svc-prod-us-east-1-vpc", modulePath, tt.outputs)
			script := RenderOperation(op)

			// First command is always cd
			if script[0] != "cd "+modulePath {
				t.Errorf("first command = %q, want cd %s", script[0], modulePath)
			}

			hasInit := false
			hasTee := false
			hasShowJSON := false
			hasSimplePlan := false
			for _, cmd := range script {
				if cmd == "terraform init" {
					hasInit = true
				}
				if contains(cmd, "tee plan.txt") {
					hasTee = true
				}
				if contains(cmd, "show -json") {
					hasShowJSON = true
				}
				if cmd == "terraform plan -out=plan.tfplan" {
					hasSimplePlan = true
				}
			}

			if hasInit != tt.wantInitCmd {
				t.Errorf("has init = %v, want %v", hasInit, tt.wantInitCmd)
			}
			if tt.wantDetailedCmds {
				if !hasTee {
					t.Error("expected tee command in detailed plan")
				}
				if !hasShowJSON {
					t.Error("expected show -json command in detailed plan")
				}
			}
			if tt.wantSimplePlan && !hasSimplePlan {
				t.Error("expected simple plan command")
			}

			if len(artifact.Paths) != tt.wantArtifactCount {
				t.Errorf("artifact count = %d, want %d", len(artifact.Paths), tt.wantArtifactCount)
			}
			// First artifact is always plan.tfplan
			if artifact.Paths[0] != modulePath+"/plan.tfplan" {
				t.Errorf("first artifact = %q, want %s/plan.tfplan", artifact.Paths[0], modulePath)
			}
			if tt.wantDetailedCmds {
				if artifact.Paths[1] != modulePath+"/plan.txt" {
					t.Errorf("second artifact = %q, want %s/plan.txt", artifact.Paths[1], modulePath)
				}
				if artifact.Paths[2] != modulePath+"/plan.json" {
					t.Errorf("third artifact = %q, want %s/plan.json", artifact.Paths[2], modulePath)
				}
			}
		})
	}
}

func TestTerraformJobConfig_ApplyScript(t *testing.T) {
	t.Parallel()

	modulePath := "svc/prod/us-east-1/vpc"

	tests := []struct {
		name         string
		config       pipeline.TerraformJobConfig
		usePlanFile  bool
		wantInitCmd  bool
		wantApplyCmd string
	}{
		{
			name:         "usePlanFile applies plan.tfplan",
			config:       mustTerraformConfig(t, false, "terraform"),
			usePlanFile:  true,
			wantInitCmd:  false,
			wantApplyCmd: "terraform apply plan.tfplan",
		},
		{
			name:         "default is plain apply",
			config:       mustTerraformConfig(t, false, "terraform"),
			wantInitCmd:  false,
			wantApplyCmd: "terraform apply",
		},
		{
			name:         "InitEnabled adds init command",
			config:       mustTerraformConfig(t, true, "terraform"),
			usePlanFile:  true,
			wantInitCmd:  true,
			wantApplyCmd: "terraform apply plan.tfplan",
		},
		{
			name:         "binary is rendered from operation",
			config:       mustTerraformConfig(t, true, "tofu"),
			usePlanFile:  true,
			wantInitCmd:  true,
			wantApplyCmd: "tofu apply plan.tfplan",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			script := RenderOperation(tt.config.NewApplyOperation(modulePath, tt.usePlanFile))

			// First command is always cd
			if script[0] != "cd "+modulePath {
				t.Errorf("first command = %q, want cd %s", script[0], modulePath)
			}

			hasInit := false
			lastCmd := script[len(script)-1]
			for _, cmd := range script {
				if cmd == tt.config.NewApplyOperation(modulePath, tt.usePlanFile).Terraform().Binary()+" init" {
					hasInit = true
				}
			}

			if hasInit != tt.wantInitCmd {
				t.Errorf("has init = %v, want %v", hasInit, tt.wantInitCmd)
			}
			if lastCmd != tt.wantApplyCmd {
				t.Errorf("apply command = %q, want %q", lastCmd, tt.wantApplyCmd)
			}
		})
	}
}

func mustTerraformConfig(tb testing.TB, initEnabled bool, binary string) pipeline.TerraformJobConfig {
	tb.Helper()
	config, err := pipeline.NewTerraformJobConfig(pipeline.TerraformJobConfigOptions{
		Binary:      binary,
		InitEnabled: initEnabled,
	})
	if err != nil {
		tb.Fatalf("NewTerraformJobConfig() error = %v", err)
	}
	return config
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || s != "" && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
