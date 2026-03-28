package ci

import "testing"

func TestPlanResultCollection_ToModulePlans(t *testing.T) {
	collection := &PlanResultCollection{
		Results: []PlanResult{
			{
				ModuleID:   "svc/prod/eu/vpc",
				ModulePath: "svc/prod/eu/vpc",
				Status:     PlanStatusChanges,
				Summary:    "+1",
			},
		},
	}

	plans := collection.ToModulePlans()
	if len(plans) != 1 {
		t.Fatalf("expected 1 plan, got %d", len(plans))
	}
	if plans[0].Status != PlanStatusChanges {
		t.Errorf("expected status changes, got %s", plans[0].Status)
	}
	if plans[0].ModuleID != "svc/prod/eu/vpc" {
		t.Errorf("expected module ID svc/prod/eu/vpc, got %s", plans[0].ModuleID)
	}
}

func TestPlanResultCollection_ToModulePlans_Empty(t *testing.T) {
	collection := &PlanResultCollection{}

	plans := collection.ToModulePlans()
	if len(plans) != 0 {
		t.Fatalf("expected 0 plans, got %d", len(plans))
	}
}

func TestPlanResultCollection_ToModulePlans_PreservesCostFields(t *testing.T) {
	collection := &PlanResultCollection{
		Results: []PlanResult{
			{
				ModuleID:   "svc/prod/eu/vpc",
				ModulePath: "svc/prod/eu/vpc",
				Status:     PlanStatusChanges,
				CostBefore: 10.0,
				CostAfter:  20.0,
				CostDiff:   10.0,
				HasCost:    true,
			},
		},
	}

	plans := collection.ToModulePlans()
	if !plans[0].HasCost {
		t.Error("expected HasCost to be true")
	}
	if plans[0].CostBefore != 10.0 {
		t.Errorf("CostBefore = %f, want 10.0", plans[0].CostBefore)
	}
	if plans[0].CostAfter != 20.0 {
		t.Errorf("CostAfter = %f, want 20.0", plans[0].CostAfter)
	}
	if plans[0].CostDiff != 10.0 {
		t.Errorf("CostDiff = %f, want 10.0", plans[0].CostDiff)
	}
}

func TestModulePlan_Get(t *testing.T) {
	plan := &ModulePlan{
		Components: map[string]string{
			"service":     "payments",
			"environment": "prod",
		},
	}

	if got := plan.Get("service"); got != "payments" {
		t.Errorf("Get(service) = %q, want payments", got)
	}
	if got := plan.Get("missing"); got != "" {
		t.Errorf("Get(missing) = %q, want empty", got)
	}
}

func TestModulePlan_Get_NilComponents(t *testing.T) {
	plan := &ModulePlan{}
	if got := plan.Get("anything"); got != "" {
		t.Errorf("Get on nil components = %q, want empty", got)
	}
}

func TestPlanResult_Get(t *testing.T) {
	result := &PlanResult{
		Components: map[string]string{
			"region": "eu-west-1",
		},
	}

	if got := result.Get("region"); got != "eu-west-1" {
		t.Errorf("Get(region) = %q, want eu-west-1", got)
	}
	if got := result.Get("missing"); got != "" {
		t.Errorf("Get(missing) = %q, want empty", got)
	}
}

func TestPlanResult_Get_NilComponents(t *testing.T) {
	result := &PlanResult{}
	if got := result.Get("anything"); got != "" {
		t.Errorf("Get on nil components = %q, want empty", got)
	}
}

func TestPlanStatus_Constants(t *testing.T) {
	tests := []struct {
		status PlanStatus
		want   string
	}{
		{PlanStatusPending, "pending"},
		{PlanStatusRunning, "running"},
		{PlanStatusSuccess, "success"},
		{PlanStatusNoChanges, "no_changes"},
		{PlanStatusChanges, "changes"},
		{PlanStatusFailed, "failed"},
	}

	for _, tt := range tests {
		if string(tt.status) != tt.want {
			t.Errorf("PlanStatus = %q, want %q", tt.status, tt.want)
		}
	}
}
