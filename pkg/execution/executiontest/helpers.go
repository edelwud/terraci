package executiontest

import (
	"testing"

	"github.com/edelwud/terraci/pkg/execution"
)

func MustJobResult(tb testing.TB, opts execution.JobResultOptions) execution.JobResult {
	tb.Helper()
	result, err := execution.NewJobResult(opts)
	if err != nil {
		tb.Fatalf("NewJobResult() error = %v", err)
	}
	return result
}

func MustGroupResult(tb testing.TB, opts execution.GroupResultOptions) execution.GroupResult {
	tb.Helper()
	result, err := execution.NewGroupResult(opts)
	if err != nil {
		tb.Fatalf("NewGroupResult() error = %v", err)
	}
	return result
}

func MustResult(tb testing.TB, opts execution.ResultOptions) *execution.Result {
	tb.Helper()
	result, err := execution.NewResult(opts)
	if err != nil {
		tb.Fatalf("NewResult() error = %v", err)
	}
	return result
}
