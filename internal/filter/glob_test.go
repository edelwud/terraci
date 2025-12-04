package filter

import (
	"testing"

	"github.com/terraci/terraci/internal/discovery"
)

func TestGlobFilter_Match(t *testing.T) {
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
			moduleID: "cdp/stage/eu-central-1/vpc",
			want:     true,
		},
		{
			name:     "exact exclude match",
			exclude:  []string{"cdp/stage/eu-central-1/vpc"},
			include:  nil,
			moduleID: "cdp/stage/eu-central-1/vpc",
			want:     false,
		},
		{
			name:     "wildcard exclude - all regions",
			exclude:  []string{"cdp/*/eu-north-1/*"},
			include:  nil,
			moduleID: "cdp/stage/eu-north-1/vpc",
			want:     false,
		},
		{
			name:     "wildcard exclude - different region passes",
			exclude:  []string{"cdp/*/eu-north-1/*"},
			include:  nil,
			moduleID: "cdp/stage/eu-central-1/vpc",
			want:     true,
		},
		{
			name:     "include only specific service",
			exclude:  nil,
			include:  []string{"cdp/*/*/*/*"},
			moduleID: "other/stage/eu-central-1/vpc",
			want:     false,
		},
		{
			name:     "include only specific service - matches",
			exclude:  nil,
			include:  []string{"cdp/*/*/*"},
			moduleID: "cdp/stage/eu-central-1/vpc",
			want:     true,
		},
		{
			name:     "exclude takes precedence",
			exclude:  []string{"cdp/stage/*/*"},
			include:  []string{"cdp/*/*/*"},
			moduleID: "cdp/stage/eu-central-1/vpc",
			want:     false,
		},
		{
			name:     "wildcard module name",
			exclude:  []string{"*/*/eu-north-1/*"},
			include:  nil,
			moduleID: "any/env/eu-north-1/module",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewGlobFilter(tt.exclude, tt.include)
			got := f.Match(tt.moduleID)
			if got != tt.want {
				t.Errorf("GlobFilter.Match() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGlobFilter_FilterModules(t *testing.T) {
	modules := []*discovery.Module{
		{Service: "cdp", Environment: "stage", Region: "eu-central-1", Module: "vpc"},
		{Service: "cdp", Environment: "stage", Region: "eu-north-1", Module: "vpc"},
		{Service: "cdp", Environment: "prod", Region: "eu-central-1", Module: "vpc"},
		{Service: "other", Environment: "stage", Region: "eu-central-1", Module: "vpc"},
	}

	f := NewGlobFilter([]string{"cdp/*/eu-north-1/*"}, nil)
	filtered := f.FilterModules(modules)

	if len(filtered) != 3 {
		t.Errorf("Expected 3 modules after filter, got %d", len(filtered))
	}

	// Verify eu-north-1 is excluded
	for _, m := range filtered {
		if m.Region == "eu-north-1" {
			t.Error("eu-north-1 should be excluded")
		}
	}
}

func TestServiceFilter(t *testing.T) {
	module := &discovery.Module{Service: "cdp", Environment: "stage", Region: "eu-central-1", Module: "vpc"}

	tests := []struct {
		services []string
		want     bool
	}{
		{nil, true},
		{[]string{"cdp"}, true},
		{[]string{"other"}, false},
		{[]string{"cdp", "other"}, true},
	}

	for _, tt := range tests {
		f := &ServiceFilter{Services: tt.services}
		if got := f.Match(module); got != tt.want {
			t.Errorf("ServiceFilter(%v).Match() = %v, want %v", tt.services, got, tt.want)
		}
	}
}

func TestEnvironmentFilter(t *testing.T) {
	module := &discovery.Module{Service: "cdp", Environment: "stage", Region: "eu-central-1", Module: "vpc"}

	tests := []struct {
		environments []string
		want         bool
	}{
		{nil, true},
		{[]string{"stage"}, true},
		{[]string{"prod"}, false},
		{[]string{"stage", "prod"}, true},
	}

	for _, tt := range tests {
		f := &EnvironmentFilter{Environments: tt.environments}
		if got := f.Match(module); got != tt.want {
			t.Errorf("EnvironmentFilter(%v).Match() = %v, want %v", tt.environments, got, tt.want)
		}
	}
}

func TestRegionFilter(t *testing.T) {
	module := &discovery.Module{Service: "cdp", Environment: "stage", Region: "eu-central-1", Module: "vpc"}

	tests := []struct {
		regions []string
		want    bool
	}{
		{nil, true},
		{[]string{"eu-central-1"}, true},
		{[]string{"us-east-1"}, false},
		{[]string{"eu-central-1", "us-east-1"}, true},
	}

	for _, tt := range tests {
		f := &RegionFilter{Regions: tt.regions}
		if got := f.Match(module); got != tt.want {
			t.Errorf("RegionFilter(%v).Match() = %v, want %v", tt.regions, got, tt.want)
		}
	}
}

func TestCompositeFilter(t *testing.T) {
	modules := []*discovery.Module{
		{Service: "cdp", Environment: "stage", Region: "eu-central-1", Module: "vpc"},
		{Service: "cdp", Environment: "prod", Region: "eu-central-1", Module: "vpc"},
		{Service: "other", Environment: "stage", Region: "eu-central-1", Module: "vpc"},
		{Service: "cdp", Environment: "stage", Region: "us-east-1", Module: "vpc"},
	}

	// Filter: service=cdp AND environment=stage
	f := NewCompositeFilter(
		&ServiceFilter{Services: []string{"cdp"}},
		&EnvironmentFilter{Environments: []string{"stage"}},
	)

	filtered := f.FilterModules(modules)

	if len(filtered) != 2 {
		t.Errorf("Expected 2 modules, got %d", len(filtered))
	}

	for _, m := range filtered {
		if m.Service != "cdp" || m.Environment != "stage" {
			t.Errorf("Unexpected module: %s/%s", m.Service, m.Environment)
		}
	}
}

func TestDoubleStarGlob(t *testing.T) {
	tests := []struct {
		pattern string
		path    string
		want    bool
	}{
		{"cdp/**", "cdp/stage/eu-central-1/vpc", true},
		{"cdp/**", "other/stage/eu-central-1/vpc", false},
		{"**/vpc", "cdp/stage/eu-central-1/vpc", true},
		{"**/vpc", "cdp/stage/eu-central-1/eks", false},
		{"cdp/**/vpc", "cdp/stage/eu-central-1/vpc", true},
		{"cdp/**/vpc", "cdp/vpc", true},
	}

	for _, tt := range tests {
		got := matchGlob(tt.pattern, tt.path)
		if got != tt.want {
			t.Errorf("matchGlob(%q, %q) = %v, want %v", tt.pattern, tt.path, got, tt.want)
		}
	}
}
