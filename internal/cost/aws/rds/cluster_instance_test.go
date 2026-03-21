package rds

import (
	"testing"
)

func TestClusterInstanceHandler_BuildLookup(t *testing.T) {
	h := &ClusterInstanceHandler{}

	tests := []struct {
		name       string
		attrs      map[string]any
		wantErr    bool
		wantClass  string
		wantEngine string
	}{
		{
			name: "aurora-mysql instance",
			attrs: map[string]any{
				"instance_class": "db.r5.large",
				"engine":         "aurora-mysql",
			},
			wantClass:  "db.r5.large",
			wantEngine: "Aurora MySQL",
		},
		{
			name: "aurora-postgresql instance",
			attrs: map[string]any{
				"instance_class": "db.r5.xlarge",
				"engine":         "aurora-postgresql",
			},
			wantClass:  "db.r5.xlarge",
			wantEngine: "Aurora PostgreSQL",
		},
		{
			name:    "missing instance_class",
			attrs:   map[string]any{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lookup, err := h.BuildLookup("us-east-1", tt.attrs)

			if tt.wantErr {
				if err == nil {
					t.Error("BuildLookup should return error")
				}
				return
			}

			if err != nil {
				t.Fatalf("BuildLookup returned error: %v", err)
			}

			if lookup.Attributes["instanceType"] != tt.wantClass {
				t.Errorf("instanceType = %q, want %q", lookup.Attributes["instanceType"], tt.wantClass)
			}
			if lookup.Attributes["databaseEngine"] != tt.wantEngine {
				t.Errorf("databaseEngine = %q, want %q", lookup.Attributes["databaseEngine"], tt.wantEngine)
			}
		})
	}
}
