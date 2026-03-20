package config

import (
	"testing"
)

func TestParsePattern(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		want    PatternSegments
		wantErr bool
	}{
		{
			name:    "default pattern",
			pattern: "{service}/{environment}/{region}/{module}",
			want:    PatternSegments{"service", "environment", "region", "module"},
		},
		{
			name:    "swapped order",
			pattern: "{environment}/{service}/{region}/{module}",
			want:    PatternSegments{"environment", "service", "region", "module"},
		},
		{
			name:    "fewer segments",
			pattern: "{team}/{module}",
			want:    PatternSegments{"team", "module"},
		},
		{
			name:    "custom names",
			pattern: "{project}/{region}/{component}",
			want:    PatternSegments{"project", "region", "component"},
		},
		{
			name:    "single segment",
			pattern: "{module}",
			want:    PatternSegments{"module"},
		},
		{
			name:    "empty pattern",
			pattern: "",
			wantErr: true,
		},
		{
			name:    "missing braces",
			pattern: "service/{environment}/{region}/{module}",
			wantErr: true,
		},
		{
			name:    "empty placeholder",
			pattern: "{}/{module}",
			wantErr: true,
		},
		{
			name:    "duplicate placeholder",
			pattern: "{service}/{service}/{module}",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParsePattern(tt.pattern)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("len = %d, want %d", len(got), len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("segment[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestPatternSegments_Contains(t *testing.T) {
	ps := PatternSegments{"service", "environment", "region", "module"}

	if !ps.Contains("service") {
		t.Error("should contain 'service'")
	}
	if !ps.Contains("module") {
		t.Error("should contain 'module'")
	}
	if ps.Contains("submodule") {
		t.Error("should not contain 'submodule'")
	}
}

func TestPatternSegments_IndexOf(t *testing.T) {
	ps := PatternSegments{"service", "environment", "region", "module"}

	if ps.IndexOf("service") != 0 {
		t.Errorf("IndexOf('service') = %d, want 0", ps.IndexOf("service"))
	}
	if ps.IndexOf("module") != 3 {
		t.Errorf("IndexOf('module') = %d, want 3", ps.IndexOf("module"))
	}
	if ps.IndexOf("unknown") != -1 {
		t.Errorf("IndexOf('unknown') = %d, want -1", ps.IndexOf("unknown"))
	}
}

func TestPatternSegments_LeafName(t *testing.T) {
	ps := PatternSegments{"service", "environment", "region", "module"}
	if ps.LeafName() != "module" {
		t.Errorf("LeafName = %q, want 'module'", ps.LeafName())
	}

	empty := PatternSegments{}
	if empty.LeafName() != "" {
		t.Errorf("empty LeafName = %q, want ''", empty.LeafName())
	}
}

func TestPatternSegments_ContextNames(t *testing.T) {
	ps := PatternSegments{"service", "environment", "region", "module"}
	ctx := ps.ContextNames()
	if len(ctx) != 3 {
		t.Fatalf("len = %d, want 3", len(ctx))
	}
	if ctx[0] != "service" || ctx[1] != "environment" || ctx[2] != "region" {
		t.Errorf("ContextNames = %v, want [service environment region]", ctx)
	}

	single := PatternSegments{"module"}
	if single.ContextNames() != nil {
		t.Errorf("single segment ContextNames should be nil, got %v", single.ContextNames())
	}
}
