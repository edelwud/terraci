package engine_test

import (
	"testing"

	"github.com/edelwud/terraci/plugins/cost/internal/engine"
	"github.com/edelwud/terraci/plugins/cost/internal/enginetest"
)

func TestTerraformPlanAdapter_LoadModule_MapsPlanToInputModel(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	enginetest.WritePlan(t, dir, enginetest.LoadPlanFixture(t, "update_ec2"))

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
		{name: "create", planJSON: enginetest.LoadPlanFixture(t, "create_ec2"), want: engine.ActionCreate},
		{name: "delete", planJSON: enginetest.LoadPlanFixture(t, "delete_ec2"), want: engine.ActionDelete},
		{name: "update", planJSON: enginetest.LoadPlanFixture(t, "update_ec2"), want: engine.ActionUpdate},
		{name: "replace", planJSON: planReplaceEC2, want: engine.ActionReplace},
		{name: "no-op", planJSON: enginetest.LoadPlanFixture(t, "no_op"), want: engine.ActionNoOp},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			enginetest.WritePlan(t, dir, tt.planJSON)

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
