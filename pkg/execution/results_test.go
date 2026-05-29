package execution

import (
	"errors"
	"reflect"
	"testing"
	"time"
)

func TestJobResultValidationAndDefensiveCopies(t *testing.T) {
	t.Parallel()

	if _, err := NewJobResult(JobResultOptions{Status: JobStatusSucceeded}); err == nil {
		t.Fatal("NewJobResult() error = nil, want missing name")
	}
	if _, err := NewJobResult(JobResultOptions{Name: "plan", Status: JobStatus("bad")}); err == nil {
		t.Fatal("NewJobResult() error = nil, want invalid status")
	}

	artifacts := []string{"plan.json"}
	result, err := NewJobResult(JobResultOptions{Name: "plan", Status: JobStatusSucceeded, ArtifactIDs: artifacts})
	if err != nil {
		t.Fatalf("NewJobResult() error = %v", err)
	}
	artifacts[0] = "mutated"
	got := result.ArtifactIDs()
	got[0] = "also-mutated"
	if want := []string{"plan.json"}; !reflect.DeepEqual(result.ArtifactIDs(), want) {
		t.Fatalf("ArtifactIDs() = %#v, want %#v", result.ArtifactIDs(), want)
	}
}

func TestResultStatsFailedAndDefensiveCopies(t *testing.T) {
	t.Parallel()

	started := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	cause := errors.New("boom")
	failed, err := NewJobResult(JobResultOptions{
		Name:       "apply",
		Status:     JobStatusFailed,
		StartedAt:  started,
		FinishedAt: started.Add(2 * time.Second),
		Err:        cause,
	})
	if err != nil {
		t.Fatalf("NewJobResult() error = %v", err)
	}
	passed, err := NewJobResult(JobResultOptions{
		Name:       "plan",
		Status:     JobStatusSucceeded,
		StartedAt:  started,
		FinishedAt: started.Add(time.Second),
	})
	if err != nil {
		t.Fatalf("NewJobResult() error = %v", err)
	}
	group, err := NewGroupResult(GroupResultOptions{Name: "dag-level-0", JobCount: 2})
	if err != nil {
		t.Fatalf("NewGroupResult() error = %v", err)
	}

	result, err := NewResult(ResultOptions{Groups: []GroupResult{group}, Jobs: []JobResult{passed, failed}})
	if err != nil {
		t.Fatalf("NewResult() error = %v", err)
	}

	stats := result.Stats()
	if stats.Groups() != 1 || stats.Jobs() != 2 || stats.Succeeded() != 1 || stats.Failed() != 1 || stats.Duration() != 3*time.Second {
		t.Fatalf("Stats() = %#v", stats)
	}
	gotFailed, ok := result.Failed()
	if !ok || gotFailed.Name() != "apply" || !errors.Is(gotFailed.Err(), cause) {
		t.Fatalf("Failed() = %#v, %v; want apply failure", gotFailed, ok)
	}

	jobs := result.Jobs()
	jobs[0] = failed
	if got := result.Jobs()[0].Name(); got != "plan" {
		t.Fatalf("Jobs() leaked mutation, first job = %q", got)
	}
}
