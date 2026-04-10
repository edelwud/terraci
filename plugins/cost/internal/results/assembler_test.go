package results_test

import (
	"errors"
	"testing"
	"time"

	"github.com/edelwud/terraci/plugins/cost/internal/model"
	"github.com/edelwud/terraci/plugins/cost/internal/results"
)

func TestAggregateCost(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		action     model.EstimateAction
		cost       float64
		wantBefore float64
		wantAfter  float64
	}{
		{"create", model.ActionCreate, 10, 0, 10},
		{"delete", model.ActionDelete, 10, 10, 0},
		{"update", model.ActionUpdate, 10, 10, 10},
		{"replace", model.ActionReplace, 10, 10, 10},
		{"no-op", model.ActionNoOp, 10, 10, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			module := &model.ModuleCost{}
			rc := model.ResourceCost{
				MonthlyCost:       tt.cost,
				BeforeMonthlyCost: tt.cost,
				Status:            model.ResourceEstimateStatusExact,
			}
			results.AggregateCost(module, rc, tt.action)

			if module.BeforeCost != tt.wantBefore {
				t.Fatalf("BeforeCost = %.2f, want %.2f", module.BeforeCost, tt.wantBefore)
			}
			if module.AfterCost != tt.wantAfter {
				t.Fatalf("AfterCost = %.2f, want %.2f", module.AfterCost, tt.wantAfter)
			}
		})
	}
}

func TestAggregateCost_UnsupportedAndUsageBased(t *testing.T) {
	t.Parallel()

	module := &model.ModuleCost{}
	results.AggregateCost(module, model.ResourceCost{MonthlyCost: 100, Status: model.ResourceEstimateStatusUnsupported}, model.ActionCreate)
	if module.AfterCost != 0 {
		t.Fatalf("AfterCost = %.2f, want 0", module.AfterCost)
	}
	if module.Unsupported != 1 {
		t.Fatalf("Unsupported = %d, want 1", module.Unsupported)
	}

	results.AggregateCost(module, model.ResourceCost{Status: model.ResourceEstimateStatusUsageUnknown}, model.ActionCreate)
	if module.Unsupported != 1 {
		t.Fatalf("Unsupported after usage-based = %d, want 1", module.Unsupported)
	}
	if module.UsageUnknown != 1 {
		t.Fatalf("UsageUnknown = %d, want 1", module.UsageUnknown)
	}
}

func TestModuleAssembler_BuildDeterministicProvidersAndSubmodules(t *testing.T) {
	t.Parallel()

	assembler := results.NewModuleAssembler(results.ModuleIdentity{
		ModuleID:   "service/prod/us-east-1/app",
		ModulePath: "/tmp/app",
		Region:     "us-east-1",
		HasChanges: true,
	})
	assembler.AddResource(model.ResourceCost{
		Provider:    "zeta",
		Address:     "module.compute.aws_instance.web",
		ModuleAddr:  "module.compute",
		MonthlyCost: 10,
	}, model.ActionCreate)
	assembler.AddResource(model.ResourceCost{
		Provider:    "alpha",
		Address:     "module.network.aws_vpc.main",
		ModuleAddr:  "module.network",
		MonthlyCost: 5,
	}, model.ActionCreate)

	module := assembler.Build()
	if module.Provider != "" {
		t.Fatalf("Provider = %q, want empty for multi-provider module", module.Provider)
	}
	if len(module.Providers) != 2 || module.Providers[0] != "alpha" || module.Providers[1] != "zeta" {
		t.Fatalf("Providers = %v, want [alpha zeta]", module.Providers)
	}
	if len(module.Submodules) != 2 {
		t.Fatalf("Submodules len = %d, want 2", len(module.Submodules))
	}
}

func TestEstimateAssembler_BuildDeterministicProvidersAndErrors(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 30, 12, 0, 0, 0, time.UTC)
	assembler := results.NewEstimateAssembler(map[string]model.ProviderMetadata{
		"zeta":  {DisplayName: "Zeta"},
		"alpha": {DisplayName: "Alpha"},
	}, now)
	assembler.AddModule(model.ModuleCost{
		ModuleID:   "mod-a",
		BeforeCost: 10,
		AfterCost:  20,
		Providers:  []string{"zeta", "alpha"},
	})
	assembler.AddModule(model.ModuleCost{
		ModuleID: "mod-b",
		Error:    "boom",
	})

	result := assembler.Build()
	if result.TotalBefore != 10 {
		t.Fatalf("TotalBefore = %.2f, want 10", result.TotalBefore)
	}
	if result.TotalAfter != 20 {
		t.Fatalf("TotalAfter = %.2f, want 20", result.TotalAfter)
	}
	if result.TotalDiff != 10 {
		t.Fatalf("TotalDiff = %.2f, want 10", result.TotalDiff)
	}
	if len(result.Providers) != 2 || result.Providers[0] != "alpha" || result.Providers[1] != "zeta" {
		t.Fatalf("Providers = %v, want [alpha zeta]", result.Providers)
	}
	if len(result.Errors) != 1 || result.Errors[0].ModuleID != "mod-b" {
		t.Fatalf("Errors = %#v, want one error for mod-b", result.Errors)
	}
	if !result.GeneratedAt.Equal(now) {
		t.Fatalf("GeneratedAt = %v, want %v", result.GeneratedAt, now)
	}
}

func TestNewErroredModule(t *testing.T) {
	t.Parallel()

	module := results.NewErroredModule("/tmp/service/prod/us-east-1/app", "us-east-1", errors.New("invalid plan"))
	if module.ModuleID != "/tmp/service/prod/us-east-1/app" {
		t.Fatalf("ModuleID = %q, want normalized path", module.ModuleID)
	}
	if module.Error != "invalid plan" {
		t.Fatalf("Error = %q, want %q", module.Error, "invalid plan")
	}
}
