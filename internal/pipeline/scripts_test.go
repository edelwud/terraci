package pipeline

import (
	"testing"
)

func TestScriptConfig_PlanScript(t *testing.T) {
	t.Parallel()

	modulePath := "svc/prod/us-east-1/vpc"

	tests := []struct {
		name              string
		config            ScriptConfig
		wantInitCmd       bool
		wantDetailedCmds  bool
		wantSimplePlan    bool
		wantArtifactCount int
	}{
		{
			name: "InitEnabled adds init command",
			config: ScriptConfig{
				TerraformBinary: "terraform",
				InitEnabled:     true,
				DetailedPlan:    false,
			},
			wantInitCmd:       true,
			wantSimplePlan:    true,
			wantArtifactCount: 1,
		},
		{
			name: "InitEnabled false skips init",
			config: ScriptConfig{
				TerraformBinary: "terraform",
				InitEnabled:     false,
				DetailedPlan:    false,
			},
			wantInitCmd:       false,
			wantSimplePlan:    true,
			wantArtifactCount: 1,
		},
		{
			name: "DetailedPlan adds tee show json commands",
			config: ScriptConfig{
				TerraformBinary: "terraform",
				InitEnabled:     false,
				DetailedPlan:    true,
			},
			wantInitCmd:       false,
			wantDetailedCmds:  true,
			wantArtifactCount: 3,
		},
		{
			name: "DetailedPlan with init",
			config: ScriptConfig{
				TerraformBinary: "terraform",
				InitEnabled:     true,
				DetailedPlan:    true,
			},
			wantInitCmd:       true,
			wantDetailedCmds:  true,
			wantArtifactCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			script, artifacts := tt.config.PlanScript(modulePath)

			// First command is always cd
			if script[0] != "cd "+modulePath {
				t.Errorf("first command = %q, want cd %s", script[0], modulePath)
			}

			hasInit := false
			hasTee := false
			hasShowJSON := false
			hasSimplePlan := false
			for _, cmd := range script {
				if cmd == "${TERRAFORM_BINARY} init" {
					hasInit = true
				}
				if contains(cmd, "tee plan.txt") {
					hasTee = true
				}
				if contains(cmd, "show -json") {
					hasShowJSON = true
				}
				if cmd == "${TERRAFORM_BINARY} plan -out=plan.tfplan" {
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

			if len(artifacts) != tt.wantArtifactCount {
				t.Errorf("artifact count = %d, want %d", len(artifacts), tt.wantArtifactCount)
			}
			// First artifact is always plan.tfplan
			if artifacts[0] != modulePath+"/plan.tfplan" {
				t.Errorf("first artifact = %q, want %s/plan.tfplan", artifacts[0], modulePath)
			}
			if tt.wantDetailedCmds {
				if artifacts[1] != modulePath+"/plan.txt" {
					t.Errorf("second artifact = %q, want %s/plan.txt", artifacts[1], modulePath)
				}
				if artifacts[2] != modulePath+"/plan.json" {
					t.Errorf("third artifact = %q, want %s/plan.json", artifacts[2], modulePath)
				}
			}
		})
	}
}

func TestScriptConfig_ApplyScript(t *testing.T) {
	t.Parallel()

	modulePath := "svc/prod/us-east-1/vpc"

	tests := []struct {
		name         string
		config       ScriptConfig
		wantInitCmd  bool
		wantApplyCmd string
	}{
		{
			name: "PlanEnabled applies plan.tfplan",
			config: ScriptConfig{
				TerraformBinary: "terraform",
				PlanEnabled:     true,
				AutoApprove:     false,
				InitEnabled:     false,
			},
			wantInitCmd:  false,
			wantApplyCmd: "${TERRAFORM_BINARY} apply plan.tfplan",
		},
		{
			name: "AutoApprove without plan uses -auto-approve",
			config: ScriptConfig{
				TerraformBinary: "terraform",
				PlanEnabled:     false,
				AutoApprove:     true,
				InitEnabled:     false,
			},
			wantInitCmd:  false,
			wantApplyCmd: "${TERRAFORM_BINARY} apply -auto-approve",
		},
		{
			name: "default is plain apply",
			config: ScriptConfig{
				TerraformBinary: "terraform",
				PlanEnabled:     false,
				AutoApprove:     false,
				InitEnabled:     false,
			},
			wantInitCmd:  false,
			wantApplyCmd: "${TERRAFORM_BINARY} apply",
		},
		{
			name: "InitEnabled adds init command",
			config: ScriptConfig{
				TerraformBinary: "terraform",
				PlanEnabled:     true,
				InitEnabled:     true,
			},
			wantInitCmd:  true,
			wantApplyCmd: "${TERRAFORM_BINARY} apply plan.tfplan",
		},
		{
			name: "PlanEnabled takes priority over AutoApprove",
			config: ScriptConfig{
				TerraformBinary: "terraform",
				PlanEnabled:     true,
				AutoApprove:     true,
				InitEnabled:     false,
			},
			wantInitCmd:  false,
			wantApplyCmd: "${TERRAFORM_BINARY} apply plan.tfplan",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			script := tt.config.ApplyScript(modulePath)

			// First command is always cd
			if script[0] != "cd "+modulePath {
				t.Errorf("first command = %q, want cd %s", script[0], modulePath)
			}

			hasInit := false
			lastCmd := script[len(script)-1]
			for _, cmd := range script {
				if cmd == "${TERRAFORM_BINARY} init" {
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
