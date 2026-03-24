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

			f := NewGlobFilter(tt.exclude, tt.include)
			if got := f.Match(tt.moduleID); got != tt.want {
				t.Errorf("Match(%q) = %v, want %v", tt.moduleID, got, tt.want)
			}
		})
	}
}

func TestGlobFilter_FilterModuleIDs(t *testing.T) {
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

			f := NewGlobFilter(tt.exclude, tt.include)
			got := f.FilterModuleIDs(tt.input)
			if len(got) != tt.wantLen {
				t.Errorf("FilterModuleIDs() returned %d, want %d", len(got), tt.wantLen)
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

			f := &SegmentFilter{Segment: tt.segment, Values: tt.values}
			if got := f.Match(module); got != tt.want {
				t.Errorf("SegmentFilter(%s=%v).Match() = %v, want %v", tt.segment, tt.values, got, tt.want)
			}
		})
	}
}

func TestCompositeFilter(t *testing.T) {
	t.Parallel()

	modules := []*discovery.Module{
		discovery.TestModule("platform", "stage", "eu-central-1", "vpc"),
		discovery.TestModule("platform", "prod", "eu-central-1", "vpc"),
		discovery.TestModule("other", "stage", "eu-central-1", "vpc"),
		discovery.TestModule("platform", "stage", "us-east-1", "vpc"),
	}

	f := NewCompositeFilter(
		&SegmentFilter{Segment: "service", Values: []string{"platform"}},
		&SegmentFilter{Segment: "environment", Values: []string{"stage"}},
	)

	filtered := f.FilterModules(modules)
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

			f := NewGlobFilter(tt.exclude, tt.include)
			if got := f.MatchModule(module); got != tt.want {
				t.Errorf("MatchModule() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestApply(t *testing.T) {
	t.Parallel()

	modules := []*discovery.Module{
		discovery.TestModule("platform", "stage", "eu-central-1", "vpc"),
		discovery.TestModule("platform", "prod", "eu-central-1", "vpc"),
		discovery.TestModule("other", "stage", "us-east-1", "rds"),
		discovery.TestModule("platform", "stage", "us-east-1", "eks"),
	}

	tests := []struct {
		name    string
		opts    Options
		wantLen int
	}{
		{"no filters returns all", Options{}, 4},
		{"filter by service", Options{Segments: map[string][]string{"service": {"platform"}}}, 3},
		{"filter by environment", Options{Segments: map[string][]string{"environment": {"stage"}}}, 3},
		{"filter by region", Options{Segments: map[string][]string{"region": {"eu-central-1"}}}, 2},
		{"combined service+env", Options{Segments: map[string][]string{"service": {"platform"}, "environment": {"stage"}}}, 2},
		{"with excludes", Options{Excludes: []string{"*/prod/*/*"}}, 3},
		{"with includes", Options{Includes: []string{"platform/*/*/*"}}, 3},
		{"all combined", Options{
			Segments: map[string][]string{"service": {"platform"}, "environment": {"stage"}, "region": {"eu-central-1"}},
		}, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := Apply(modules, tt.opts)
			if len(got) != tt.wantLen {
				t.Errorf("Apply() returned %d modules, want %d", len(got), tt.wantLen)
			}
		})
	}
}

func TestMatchGlob_NoDoubleStar(t *testing.T) {
	t.Parallel()

	tests := []struct {
		pattern string
		path    string
		want    bool
	}{
		{"platform/*/*/*", "platform/stage/eu-central-1/vpc", true},
		{"platform/*/*/*", "other/stage/eu-central-1/vpc", false},
		{"*", "anything", true},
	}

	for _, tt := range tests {
		if got := matchGlob(tt.pattern, tt.path); got != tt.want {
			t.Errorf("matchGlob(%q, %q) = %v, want %v", tt.pattern, tt.path, got, tt.want)
		}
	}
}

func TestMatchSegments(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		pattern string
		path    string
		prefix  bool
		want    bool
	}{
		{"prefix match", "platform/stage", "platform/stage/eu-central-1/vpc", true, true},
		{"prefix no match", "other/stage", "platform/stage/eu-central-1/vpc", true, false},
		{"prefix too long", "a/b/c/d/e", "a/b", true, false},
		{"suffix match", "vpc", "platform/stage/eu-central-1/vpc", false, true},
		{"suffix no match", "rds", "platform/stage/eu-central-1/vpc", false, false},
		{"suffix multi", "eu-central-1/vpc", "platform/stage/eu-central-1/vpc", false, true},
		{"suffix too long", "a/b/c/d/e", "a/b", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := matchSegments(tt.pattern, tt.path, tt.prefix); got != tt.want {
				t.Errorf("matchSegments(%q, %q, %v) = %v, want %v", tt.pattern, tt.path, tt.prefix, got, tt.want)
			}
		})
	}
}

func TestParseSegmentFilters(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
		want map[string][]string
	}{
		{"empty", nil, map[string][]string{}},
		{"single filter", []string{"env=prod"}, map[string][]string{"env": {"prod"}}},
		{"multiple values", []string{"env=prod", "env=stage"}, map[string][]string{"env": {"prod", "stage"}}},
		{"multiple keys", []string{"env=prod", "region=eu"}, map[string][]string{"env": {"prod"}, "region": {"eu"}}},
		{"no equals sign ignored", []string{"invalid"}, map[string][]string{}},
		{"empty key ignored", []string{"=value"}, map[string][]string{}},
		{"empty value allowed", []string{"key="}, map[string][]string{"key": {""}}},
		{"value with equals", []string{"key=a=b"}, map[string][]string{"key": {"a=b"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := ParseSegmentFilters(tt.args)
			if len(got) != len(tt.want) {
				t.Fatalf("ParseSegmentFilters(%v) = %v, want %v", tt.args, got, tt.want)
			}
			for k, wantVals := range tt.want {
				gotVals := got[k]
				if len(gotVals) != len(wantVals) {
					t.Errorf("key %q: got %v, want %v", k, gotVals, wantVals)
					continue
				}
				for i := range wantVals {
					if gotVals[i] != wantVals[i] {
						t.Errorf("key %q[%d]: got %q, want %q", k, i, gotVals[i], wantVals[i])
					}
				}
			}
		})
	}
}

func TestDoubleStarGlob(t *testing.T) {
	t.Parallel()

	tests := []struct {
		pattern string
		path    string
		want    bool
	}{
		{"platform/**", "platform/stage/eu-central-1/vpc", true},
		{"platform/**", "other/stage/eu-central-1/vpc", false},
		{"**/vpc", "platform/stage/eu-central-1/vpc", true},
		{"**/vpc", "platform/stage/eu-central-1/eks", false},
		{"platform/**/vpc", "platform/stage/eu-central-1/vpc", true},
		{"platform/**/vpc", "platform/vpc", true},
	}

	for _, tt := range tests {
		if got := matchGlob(tt.pattern, tt.path); got != tt.want {
			t.Errorf("matchGlob(%q, %q) = %v, want %v", tt.pattern, tt.path, got, tt.want)
		}
	}
}
