package ci

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestPlanResultCollection_Results(t *testing.T) {
	plan := mustPlanResult(t, PlanResultOptions{
		ModuleID:   "svc/prod/eu/vpc",
		ModulePath: "svc/prod/eu/vpc",
		Status:     PlanStatusChanges,
		Summary:    "+1",
	})
	collection := mustPlanResultCollection(t, PlanResultCollectionOptions{
		Results: []PlanResult{plan},
	})

	plans := collection.Results()
	if len(plans) != 1 {
		t.Fatalf("expected 1 plan, got %d", len(plans))
	}
	if plans[0].Status() != PlanStatusChanges {
		t.Errorf("expected status changes, got %s", plans[0].Status())
	}
	if plans[0].ModuleID() != "svc/prod/eu/vpc" {
		t.Errorf("expected module ID svc/prod/eu/vpc, got %s", plans[0].ModuleID())
	}
}

func TestPlanResultCollection_Results_Empty(t *testing.T) {
	collection := EmptyPlanResultCollection()

	if collection.Len() != 0 {
		t.Fatalf("expected 0 plans, got %d", collection.Len())
	}
	data, err := json.Marshal(collection)
	if err != nil {
		t.Fatalf("MarshalJSON() error = %v", err)
	}
	if !containsJSONFragment(data, `"results":[]`) {
		t.Fatalf("empty collection JSON = %s, want results []", data)
	}
}

func TestPlanResultCollection_FingerprintStableAcrossOrder(t *testing.T) {
	a := mustPlanResult(t, PlanResultOptions{
		ModuleID:          "svc/prod/us-east-1/vpc",
		ModulePath:        "svc/prod/us-east-1/vpc",
		Status:            PlanStatusChanges,
		Summary:           "+1",
		StructuredDetails: "details",
		RawPlanOutput:     "raw",
		ExitCode:          2,
	})
	b := mustPlanResult(t, PlanResultOptions{
		ModuleID:   "svc/prod/us-east-1/rds",
		ModulePath: "svc/prod/us-east-1/rds",
		Status:     PlanStatusNoChanges,
		Summary:    "No changes",
		ExitCode:   0,
	})

	first := mustPlanResultCollection(t, PlanResultCollectionOptions{Results: []PlanResult{a, b}}).Fingerprint()
	second := mustPlanResultCollection(t, PlanResultCollectionOptions{Results: []PlanResult{b, a}}).Fingerprint()
	if first == "" {
		t.Fatal("Fingerprint() = empty, want value")
	}
	if first != second {
		t.Fatalf("Fingerprint() should be stable across order: %q != %q", first, second)
	}
}

func TestPlanResult_Component(t *testing.T) {
	result := mustPlanResult(t, PlanResultOptions{
		ModuleID:   "svc/prod/eu/vpc",
		ModulePath: "svc/prod/eu/vpc",
		Status:     PlanStatusChanges,
		Components: map[string]string{
			"region":      "eu-west-1",
			"environment": "prod",
		},
	})

	if got := result.Component("region"); got != "eu-west-1" {
		t.Errorf("Component(region) = %q, want eu-west-1", got)
	}
	if got := result.Component("missing"); got != "" {
		t.Errorf("Component(missing) = %q, want empty", got)
	}
}

func TestPlanResult_Component_NilComponents(t *testing.T) {
	result := mustPlanResult(t, PlanResultOptions{
		ModuleID:   "svc/prod/eu/vpc",
		ModulePath: "svc/prod/eu/vpc",
		Status:     PlanStatusNoChanges,
	})
	if got := result.Component("anything"); got != "" {
		t.Errorf("Component on nil components = %q, want empty", got)
	}
}

func TestPlanResult_JSONRoundTripAndDefensiveComponents(t *testing.T) {
	result := mustPlanResult(t, PlanResultOptions{
		ModuleID:   "svc/prod/eu/vpc",
		ModulePath: "svc/prod/eu/vpc",
		Status:     PlanStatusChanges,
		Components: map[string]string{
			"environment": "prod",
		},
		Summary:           "+1",
		StructuredDetails: "details",
		RawPlanOutput:     "raw",
		ExitCode:          2,
	})

	components := result.Components()
	components["environment"] = "mutated"
	if result.Component("environment") != "prod" {
		t.Fatal("Components() returned mutable internal map")
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("MarshalJSON() error = %v", err)
	}
	var decoded PlanResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("UnmarshalJSON() error = %v", err)
	}
	if decoded.ModuleID() != result.ModuleID() ||
		decoded.ModulePath() != result.ModulePath() ||
		decoded.Status() != result.Status() ||
		decoded.Summary() != result.Summary() ||
		decoded.StructuredDetails() != result.StructuredDetails() ||
		decoded.RawPlanOutput() != result.RawPlanOutput() ||
		decoded.ExitCode() != result.ExitCode() {
		t.Fatalf("decoded = %#v, want round-trip fields", decoded)
	}
}

func TestPlanResultCollection_JSONRoundTrip(t *testing.T) {
	generatedAt := time.Date(2026, 6, 4, 10, 0, 0, 0, time.UTC)
	plan := mustPlanResult(t, PlanResultOptions{
		ModuleID:   "svc/prod/eu/vpc",
		ModulePath: "svc/prod/eu/vpc",
		Status:     PlanStatusChanges,
	})
	collection := mustPlanResultCollection(t, PlanResultCollectionOptions{
		Results:     []PlanResult{plan},
		PipelineID:  "pipeline",
		CommitSHA:   "commit",
		GeneratedAt: generatedAt,
	})

	data, err := json.Marshal(collection)
	if err != nil {
		t.Fatalf("MarshalJSON() error = %v", err)
	}
	var decoded PlanResultCollection
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("UnmarshalJSON() error = %v", err)
	}
	if decoded.PipelineID() != "pipeline" || decoded.CommitSHA() != "commit" || !decoded.GeneratedAt().Equal(generatedAt) {
		t.Fatalf("decoded metadata = (%q, %q, %v), want original metadata", decoded.PipelineID(), decoded.CommitSHA(), decoded.GeneratedAt())
	}
	if decoded.Len() != 1 || decoded.Results()[0].ModuleID() != "svc/prod/eu/vpc" {
		t.Fatalf("decoded results = %#v, want original result", decoded.Results())
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

func mustPlanResult(tb testing.TB, opts PlanResultOptions) PlanResult {
	tb.Helper()
	result, err := NewPlanResult(opts)
	if err != nil {
		tb.Fatalf("NewPlanResult() error = %v", err)
	}
	return result
}

func mustPlanResultCollection(tb testing.TB, opts PlanResultCollectionOptions) *PlanResultCollection {
	tb.Helper()
	collection, err := NewPlanResultCollection(opts)
	if err != nil {
		tb.Fatalf("NewPlanResultCollection() error = %v", err)
	}
	return collection
}

func containsJSONFragment(data []byte, fragment string) bool {
	return strings.Contains(string(data), fragment)
}
