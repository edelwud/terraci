package engine_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/edelwud/terraci/plugins/cost/internal/engine"
)

func TestTerraformPlanAdapter_LoadModule_MapsPlanToInputModel(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeTerraformPlanFixture(t, dir, planUpdateEC2)

	adapter := engine.NewTerraformPlanAdapter()
	modulePlan, err := adapter.LoadModule(dir, "us-east-1")
	if err != nil {
		t.Fatalf("LoadModule() error = %v", err)
	}

	if modulePlan.ModulePath != dir {
		t.Fatalf("ModulePath = %q, want %q", modulePlan.ModulePath, dir)
	}
	if modulePlan.Region != "us-east-1" {
		t.Fatalf("Region = %q, want us-east-1", modulePlan.Region)
	}
	if !modulePlan.HasChanges {
		t.Fatal("HasChanges = false, want true")
	}
	if len(modulePlan.Resources) != 1 {
		t.Fatalf("Resources len = %d, want 1", len(modulePlan.Resources))
	}

	resource := modulePlan.Resources[0]
	if resource.Action != engine.ActionUpdate {
		t.Fatalf("Action = %q, want %q", resource.Action, engine.ActionUpdate)
	}
	if resource.Address != "aws_instance.web" {
		t.Fatalf("Address = %q, want aws_instance.web", resource.Address)
	}
	if resource.Name != "web" {
		t.Fatalf("Name = %q, want web", resource.Name)
	}
	if resource.AfterAttrs == nil || resource.BeforeAttrs == nil {
		t.Fatal("expected both before and after attrs to be populated")
	}
	if resource.ModuleAddr != "" {
		t.Fatalf("ModuleAddr = %q, want empty root module address", resource.ModuleAddr)
	}
}

func TestTerraformPlanAdapter_LoadModule_MapsAllSupportedActions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		planJSON string
		want     engine.EstimateAction
	}{
		{name: "create", planJSON: planCreateEC2, want: engine.ActionCreate},
		{name: "delete", planJSON: planDeleteEC2, want: engine.ActionDelete},
		{name: "update", planJSON: planUpdateEC2, want: engine.ActionUpdate},
		{name: "replace", planJSON: planReplaceEC2, want: engine.ActionReplace},
		{name: "no-op", planJSON: planNoOp, want: engine.ActionNoOp},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			writeTerraformPlanFixture(t, dir, tt.planJSON)

			modulePlan, err := engine.NewTerraformPlanAdapter().LoadModule(dir, "us-east-1")
			if err != nil {
				t.Fatalf("LoadModule() error = %v", err)
			}
			if got := modulePlan.Resources[0].Action; got != tt.want {
				t.Fatalf("Action = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMapTerraformAction_RejectsUnknownAction(t *testing.T) {
	t.Parallel()

	_, err := engine.MapTerraformAction("import")
	if err == nil {
		t.Fatal("mapTerraformAction() error = nil, want error for unknown action")
	}
}

func writeTerraformPlanFixture(t *testing.T, dir, planJSON string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "plan.json"), []byte(planJSON), 0o600); err != nil {
		t.Fatal(err)
	}
}
