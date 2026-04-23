package localexec

import (
	"context"
	"testing"

	"github.com/edelwud/terraci/pkg/filter"
	"github.com/edelwud/terraci/pkg/plugin/plugintest"
)

func TestMapExecuteRequest_NilFiltersSafe(t *testing.T) {
	t.Parallel()

	req, err := mapExecuteRequest(ExecuteRequest{Mode: ExecutionModePlan})
	if err != nil {
		t.Fatalf("mapExecuteRequest() error = %v", err)
	}
	if req.Filters == nil {
		t.Fatal("mapped filters should not be nil")
	}
}

func TestMapExecuteRequest_ModeMapping(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		mode ExecutionMode
	}{
		{name: "run", mode: ExecutionModeRun},
		{name: "plan", mode: ExecutionModePlan},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req, err := mapExecuteRequest(ExecuteRequest{
				Mode:        tt.mode,
				Parallelism: -1,
				Filters:     &filter.Flags{SegmentArgs: []string{"environment=stage"}},
			})
			if err != nil {
				t.Fatalf("mapExecuteRequest() error = %v", err)
			}
			if req.Mode.String() != tt.mode.String() {
				t.Fatalf("mapped mode = %q, want %q", req.Mode.String(), tt.mode.String())
			}
			if req.Parallelism != -1 {
				t.Fatalf("parallelism = %d, want -1 passthrough", req.Parallelism)
			}
			if req.Filters == nil || len(req.Filters.SegmentArgs) != 1 {
				t.Fatalf("filters = %#v, want preserved filters", req.Filters)
			}
		})
	}
}

func TestMapExecuteRequest_InvalidMode(t *testing.T) {
	t.Parallel()

	if _, err := mapExecuteRequest(ExecuteRequest{Mode: ExecutionMode(99)}); err == nil {
		t.Fatal("mapExecuteRequest() error = nil, want invalid mode error")
	}
}

func TestExecutorRun_NilFiltersDoesNotPanic(t *testing.T) {
	t.Parallel()

	appCtx := plugintest.NewAppContext(t, t.TempDir())
	executor := NewExecutor(appCtx)

	if err := executor.Run(context.Background(), ExecuteRequest{Mode: ExecutionModePlan}); err == nil {
		t.Fatal("Run() error = nil, want workflow error without panic")
	}
}
