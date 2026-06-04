package ci

import (
	"strings"
	"testing"
	"time"
)

func TestNewArtifactRun_DerivesFingerprintFromPlanResults(t *testing.T) {
	t.Parallel()

	collection := testPlanResultCollection()
	run, err := NewArtifactRun(ArtifactRunOptions{
		Producer:    "cost",
		PlanResults: collection,
	})
	if err != nil {
		t.Fatalf("NewArtifactRun() error = %v", err)
	}
	if run.Producer() != "cost" {
		t.Fatalf("Producer = %q, want cost", run.Producer())
	}
	if run.Artifact().PlanResultsFingerprint() != collection.Fingerprint() {
		t.Fatalf("fingerprint = %q, want %q", run.Artifact().PlanResultsFingerprint(), collection.Fingerprint())
	}
	if run.Artifact().GeneratedAt().IsZero() {
		t.Fatal("GeneratedAt is zero, want default timestamp")
	}
}

func TestNewArtifactRun_PreservesExplicitFingerprint(t *testing.T) {
	t.Parallel()

	generatedAt := time.Date(2026, 5, 19, 10, 0, 0, 0, time.FixedZone("custom", 3*60*60))
	run, err := NewArtifactRun(ArtifactRunOptions{
		Producer: "policy",
		Artifact: NewArtifactContext(ArtifactContextOptions{
			PlanResultsFingerprint: "explicit",
			GeneratedAt:            generatedAt,
		}),
		PlanResults: testPlanResultCollection(),
	})
	if err != nil {
		t.Fatalf("NewArtifactRun() error = %v", err)
	}
	if run.Artifact().PlanResultsFingerprint() != "explicit" {
		t.Fatalf("fingerprint = %q, want explicit", run.Artifact().PlanResultsFingerprint())
	}
	if run.Artifact().GeneratedAt().Location() != time.UTC {
		t.Fatalf("GeneratedAt location = %v, want UTC", run.Artifact().GeneratedAt().Location())
	}
}

func TestNewArtifactRun_AllowsDegradedModeWithoutPlanResults(t *testing.T) {
	t.Parallel()

	run, err := NewArtifactRun(ArtifactRunOptions{Producer: "tfupdate"})
	if err != nil {
		t.Fatalf("NewArtifactRun() error = %v", err)
	}
	if run.Artifact().PlanResultsFingerprint() != "" {
		t.Fatalf("fingerprint = %q, want empty degraded mode", run.Artifact().PlanResultsFingerprint())
	}
}

func TestNewArtifactRun_ValidatesProducer(t *testing.T) {
	t.Parallel()

	_, err := NewArtifactRun(ArtifactRunOptions{Producer: "../bad"})
	if err == nil {
		t.Fatal("NewArtifactRun() error = nil, want invalid producer error")
	}
	if !strings.Contains(err.Error(), "safe artifact name") {
		t.Fatalf("NewArtifactRun() error = %q, want safe artifact name message", err.Error())
	}
}

func testPlanResultCollection() *PlanResultCollection {
	result, err := NewPlanResult(PlanResultOptions{
		ModuleID:   "svc/prod/us/vpc",
		ModulePath: "svc/prod/us/vpc",
		Status:     PlanStatusChanges,
		Summary:    "+1",
	})
	if err != nil {
		panic(err)
	}
	collection, err := NewPlanResultCollection(PlanResultCollectionOptions{
		Results: []PlanResult{result},
	})
	if err != nil {
		panic(err)
	}
	return collection
}
