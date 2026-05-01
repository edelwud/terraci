package ci

import "testing"

func TestPlanResultCollection_Results(t *testing.T) {
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

	if len(collection.Results) != 1 {
		t.Fatalf("expected 1 plan, got %d", len(collection.Results))
	}
	if collection.Results[0].Status != PlanStatusChanges {
		t.Errorf("expected status changes, got %s", collection.Results[0].Status)
	}
	if collection.Results[0].ModuleID != "svc/prod/eu/vpc" {
		t.Errorf("expected module ID svc/prod/eu/vpc, got %s", collection.Results[0].ModuleID)
	}
}

func TestPlanResultCollection_Results_Empty(t *testing.T) {
	collection := &PlanResultCollection{}

	if len(collection.Results) != 0 {
		t.Fatalf("expected 0 plans, got %d", len(collection.Results))
	}
}

func TestPlanResultCollection_FingerprintStableAcrossOrder(t *testing.T) {
	a := PlanResult{
		ModuleID:          "svc/prod/us-east-1/vpc",
		ModulePath:        "svc/prod/us-east-1/vpc",
		Status:            PlanStatusChanges,
		Summary:           "+1",
		StructuredDetails: "details",
		RawPlanOutput:     "raw",
		ExitCode:          2,
	}
	b := PlanResult{
		ModuleID:   "svc/prod/us-east-1/rds",
		ModulePath: "svc/prod/us-east-1/rds",
		Status:     PlanStatusNoChanges,
		Summary:    "No changes",
		ExitCode:   0,
	}

	first := (&PlanResultCollection{Results: []PlanResult{a, b}}).Fingerprint()
	second := (&PlanResultCollection{Results: []PlanResult{b, a}}).Fingerprint()
	if first == "" {
		t.Fatal("Fingerprint() = empty, want value")
	}
	if first != second {
		t.Fatalf("Fingerprint() should be stable across order: %q != %q", first, second)
	}
}

func TestPlanResult_Get(t *testing.T) {
	result := &PlanResult{
		Components: map[string]string{
			"region":      "eu-west-1",
			"environment": "prod",
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
