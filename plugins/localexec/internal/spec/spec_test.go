package spec

import (
	"testing"

	"github.com/edelwud/terraci/pkg/filter"
)

func TestNormalizeExecuteRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		req     ExecuteRequest
		wantErr bool
	}{
		{
			name: "run mode with nil filters",
			req: ExecuteRequest{
				Mode: ExecutionModeRun,
			},
		},
		{
			name: "plan mode with explicit filters",
			req: ExecuteRequest{
				Mode:    ExecutionModePlan,
				Filters: &filter.Flags{Excludes: []string{"*/test/*"}},
			},
		},
		{
			name: "invalid mode",
			req: ExecuteRequest{
				Mode: ExecutionMode(99),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := NormalizeExecuteRequest(tt.req)
			if tt.wantErr {
				if err == nil {
					t.Fatal("NormalizeExecuteRequest() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("NormalizeExecuteRequest() error = %v", err)
			}
			if got.Filters == nil {
				t.Fatal("NormalizeExecuteRequest() filters = nil, want empty filter set")
			}
		})
	}
}
