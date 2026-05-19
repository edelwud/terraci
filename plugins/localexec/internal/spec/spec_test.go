package spec

import (
	"testing"

	"github.com/edelwud/terraci/pkg/filter"
)

func TestNormalizeRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		req     Request
		wantErr bool
	}{
		{
			name: "run mode with nil filters",
			req: Request{
				Mode: ExecutionModeRun,
			},
		},
		{
			name: "plan mode with explicit filters",
			req: Request{
				Mode:    ExecutionModePlan,
				Filters: &filter.Flags{Excludes: []string{"*/test/*"}},
			},
		},
		{
			name: "invalid mode",
			req: Request{
				Mode: ExecutionMode(99),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := NormalizeRequest(tt.req)
			if tt.wantErr {
				if err == nil {
					t.Fatal("NormalizeRequest() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("NormalizeRequest() error = %v", err)
			}
			if got.Filters == nil {
				t.Fatal("NormalizeRequest() filters = nil, want empty filter set")
			}
		})
	}
}
