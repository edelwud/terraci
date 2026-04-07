package citest

import (
	"slices"
	"strings"
	"testing"

	"github.com/edelwud/terraci/pkg/pipeline"
)

func AssertHasNeed(tb testing.TB, jobName string, needs []string, expected string) {
	tb.Helper()
	if slices.Contains(needs, expected) {
		return
	}
	tb.Fatalf("expected job %q to need %q, got %v", jobName, expected, needs)
}

func AssertNoNeed(tb testing.TB, jobName string, needs []string, unexpected string) {
	tb.Helper()
	for _, need := range needs {
		if need == unexpected {
			tb.Fatalf("expected job %q to not need %q", jobName, unexpected)
		}
	}
}

func AssertNoNeedWithPrefix(tb testing.TB, jobName string, needs []string, prefix string) {
	tb.Helper()
	for _, need := range needs {
		if strings.HasPrefix(need, prefix) {
			tb.Fatalf("expected job %q to not have need with prefix %q, got %q", jobName, prefix, need)
		}
	}
}

type DryRunExpectation struct {
	TotalModules    int
	AffectedModules int
	Jobs            int
	Stages          int
	ExecutionLevels int
}

func AssertDryRun(tb testing.TB, result *pipeline.DryRunResult, expected DryRunExpectation) {
	tb.Helper()
	if result.TotalModules != expected.TotalModules {
		tb.Fatalf("expected TotalModules=%d, got %d", expected.TotalModules, result.TotalModules)
	}
	if result.AffectedModules != expected.AffectedModules {
		tb.Fatalf("expected AffectedModules=%d, got %d", expected.AffectedModules, result.AffectedModules)
	}
	if result.Jobs != expected.Jobs {
		tb.Fatalf("expected Jobs=%d, got %d", expected.Jobs, result.Jobs)
	}
	if result.Stages != expected.Stages {
		tb.Fatalf("expected Stages=%d, got %d", expected.Stages, result.Stages)
	}
	if len(result.ExecutionOrder) != expected.ExecutionLevels {
		tb.Fatalf("expected %d execution levels, got %d", expected.ExecutionLevels, len(result.ExecutionOrder))
	}
}
