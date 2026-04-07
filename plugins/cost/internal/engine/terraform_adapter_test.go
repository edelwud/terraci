package engine_test

import (
	"testing"

	"github.com/edelwud/terraci/plugins/cost/internal/engine"
	"github.com/edelwud/terraci/plugins/cost/internal/enginetest"
	"github.com/edelwud/terraci/plugins/cost/internal/model"
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
	if resource.Action != model.ActionUpdate {
		t.Fatalf("Action = %q, want %q", resource.Action, model.ActionUpdate)
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
		want     model.EstimateAction
	}{
		{name: "create", planJSON: enginetest.LoadPlanFixture(t, "create_ec2"), want: model.ActionCreate},
		{name: "delete", planJSON: enginetest.LoadPlanFixture(t, "delete_ec2"), want: model.ActionDelete},
		{name: "update", planJSON: enginetest.LoadPlanFixture(t, "update_ec2"), want: model.ActionUpdate},
		{name: "replace", planJSON: planReplaceEC2, want: model.ActionReplace},
		{name: "no-op", planJSON: enginetest.LoadPlanFixture(t, "no_op"), want: model.ActionNoOp},
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

func TestTerraformPlanAdapter_LoadModule_IgnoresReadActions(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	enginetest.WritePlan(t, dir, `{
		"format_version": "1.2",
		"terraform_version": "1.6.0",
		"resource_changes": [
			{
				"address": "data.aws_eks_addon_version.this",
				"mode": "data",
				"type": "aws_eks_addon_version",
				"name": "this",
				"change": {
					"actions": ["read"],
					"before": null,
					"after": {"addon_name": "coredns"},
					"after_unknown": {}
				}
			},
			{
				"address": "aws_instance.web",
				"module_address": "",
				"type": "aws_instance",
				"name": "web",
				"change": {
					"actions": ["create"],
					"before": null,
					"after": {"instance_type": "t3.micro", "ami": "ami-12345"},
					"after_unknown": {}
				}
			}
		]
	}`)

	modulePlan, err := engine.NewTerraformPlanAdapter().LoadModule(dir, "us-east-1")
	if err != nil {
		t.Fatalf("LoadModule() error = %v", err)
	}
	if len(modulePlan.Resources) != 1 {
		t.Fatalf("Resources len = %d, want 1 after skipping read action", len(modulePlan.Resources))
	}
	if modulePlan.Resources[0].Address != "aws_instance.web" {
		t.Fatalf("Resource[0].Address = %q, want aws_instance.web", modulePlan.Resources[0].Address)
	}
}
