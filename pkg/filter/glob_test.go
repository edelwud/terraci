package filter

import (
	"testing"

	"github.com/edelwud/terraci/pkg/discovery"
)

func TestGlobFilter_Match(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		exclude  []string
		include  []string
		moduleID string
		want     bool
	}{
		{
			name:     "no filters - include all",
			exclude:  nil,
			include:  nil,
			moduleID: "platform/stage/eu-central-1/vpc",
			want:     true,
		},
		{
			name:     "exact exclude match",
			exclude:  []string{"platform/stage/eu-central-1/vpc"},
			include:  nil,
			moduleID: "platform/stage/eu-central-1/vpc",
			want:     false,
		},
		{
			name:     "wildcard exclude",
			exclude:  []string{"platform/*/eu-central-1/vpc"},
			include:  nil,
			moduleID: "platform/stage/eu-central-1/vpc",
			want:     false,
		},
		{
			name:     "exclude doesn't match",
			exclude:  []string{"other/*/*/vpc"},
			include:  nil,
			moduleID: "platform/stage/eu-central-1/vpc",
			want:     true,
		},
		{
			name:     "double star exclude",
			exclude:  []string{"**/vpc"},
			include:  nil,
			moduleID: "platform/stage/eu-central-1/vpc",
			want:     false,
		},
		{
			name:     "include pattern matches",
			exclude:  nil,
			include:  []string{"platform/*/*/*"},
			moduleID: "platform/stage/eu-central-1/vpc",
			want:     true,
		},
		{
			name:     "include pattern doesn't match",
			exclude:  nil,
			include:  []string{"other/*/*/*"},
			moduleID: "platform/stage/eu-central-1/vpc",
			want:     false,
		},
		{
			name:     "exclude takes precedence over include",
			exclude:  []string{"**/vpc"},
			include:  []string{"platform/*/*/*"},
			moduleID: "platform/stage/eu-central-1/vpc",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := globFilter{excludes: tt.exclude, includes: tt.include}
			if got := f.matchID(tt.moduleID); got != tt.want {
				t.Errorf("matchID(%q) = %v, want %v", tt.moduleID, got, tt.want)
			}
		})
	}
}

func TestApply_GlobFilters(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		exclude []string
		include []string
		input   []string
		wantLen int
	}{
		{"no filters", nil, nil, []string{"a/b/c/d", "e/f/g/h"}, 2},
		{"exclude one", []string{"a/b/c/d"}, nil, []string{"a/b/c/d", "e/f/g/h"}, 1},
		{"include specific", nil, []string{"a/*/*/*"}, []string{"a/b/c/d", "e/f/g/h"}, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			modules := make([]*discovery.Module, len(tt.input))
			for i, id := range tt.input {
				parts := splitID(id)
				modules[i] = discovery.TestModule(parts[0], parts[1], parts[2], parts[3])
			}

			got := Apply(modules, Options{Excludes: tt.exclude, Includes: tt.include})
			if len(got) != tt.wantLen {
				t.Errorf("Apply() returned %d, want %d", len(got), tt.wantLen)
			}
		})
	}
}

func TestSegmentFilter(t *testing.T) {
	t.Parallel()

	module := discovery.TestModule("platform", "stage", "eu-central-1", "vpc")

	tests := []struct {
		name    string
		segment string
		values  []string
		want    bool
	}{
		{"empty values matches all", "service", nil, true},
		{"service match", "service", []string{"platform"}, true},
		{"service no match", "service", []string{"other"}, false},
		{"service multi match", "service", []string{"platform", "other"}, true},
		{"environment match", "environment", []string{"stage"}, true},
		{"environment no match", "environment", []string{"prod"}, false},
		{"region match", "region", []string{"eu-central-1"}, true},
		{"region no match", "region", []string{"us-east-1"}, false},
		{"region multi match", "region", []string{"eu-central-1", "us-east-1"}, true},
		{"unknown segment", "team", []string{"devops"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := segmentFilter{segment: tt.segment, values: tt.values}
			if got := f.match(module); got != tt.want {
				t.Errorf("segmentFilter(%s=%v).match() = %v, want %v", tt.segment, tt.values, got, tt.want)
			}
		})
	}
}

func TestApply_CompositeFilters(t *testing.T) {
	t.Parallel()

	modules := []*discovery.Module{
		discovery.TestModule("platform", "stage", "eu-central-1", "vpc"),
		discovery.TestModule("platform", "prod", "eu-central-1", "vpc"),
		discovery.TestModule("other", "stage", "eu-central-1", "vpc"),
		discovery.TestModule("platform", "stage", "us-east-1", "vpc"),
	}

	filtered := Apply(modules, Options{
		Segments: map[string][]string{
			"service":     {"platform"},
			"environment": {"stage"},
		},
	})
	if len(filtered) != 2 {
		t.Errorf("Expected 2 modules, got %d", len(filtered))
	}
	for _, m := range filtered {
		if m.Get("service") != "platform" || m.Get("environment") != "stage" {
			t.Errorf("Unexpected module: %s/%s", m.Get("service"), m.Get("environment"))
		}
	}
}

func TestGlobFilter_MatchModule(t *testing.T) {
	t.Parallel()

	module := discovery.TestModule("platform", "stage", "eu-central-1", "vpc")

	tests := []struct {
		name    string
		exclude []string
		include []string
		want    bool
	}{
		{"no filters matches", nil, nil, true},
		{"exclude match rejects", []string{"platform/*/*/*"}, nil, false},
		{"include match accepts", nil, []string{"platform/*/*/*"}, true},
		{"include no match rejects", nil, []string{"other/*/*/*"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := globFilter{excludes: tt.exclude, includes: tt.include}
			if got := f.match(module); got != tt.want {
				t.Errorf("match() = %v, want %v", got, tt.want)
			}
		})
	}
}

// splitID splits "a/b/c/d" into [a b c d]; used by TestApply_GlobFilters.
func splitID(id string) [4]string {
	var out [4]string
	idx := 0
	start := 0
	for i := 0; i < len(id) && idx < 4; i++ {
		if id[i] == '/' {
			out[idx] = id[start:i]
			start = i + 1
			idx++
		}
	}
	if idx < 4 {
		out[idx] = id[start:]
	}
	return out
}
