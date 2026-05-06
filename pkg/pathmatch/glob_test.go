package pathmatch

import "testing"

func TestMatchGlob(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		pattern string
		path    string
		want    bool
	}{
		{name: "exact", pattern: "platform/stage/eu-central-1/vpc", path: "platform/stage/eu-central-1/vpc", want: true},
		{name: "single segment wildcard", pattern: "platform/*/eu-central-1/vpc", path: "platform/stage/eu-central-1/vpc", want: true},
		{name: "single segment wildcard does not cross slash", pattern: "platform/*/vpc", path: "platform/stage/eu-central-1/vpc", want: false},
		{name: "globstar middle", pattern: "platform/**/vpc", path: "platform/stage/eu-central-1/vpc", want: true},
		{name: "globstar zero middle segments", pattern: "platform/**/vpc", path: "platform/vpc", want: true},
		{name: "globstar prefix", pattern: "**/sandbox/**", path: "platform/sandbox/eu-central-1/app", want: true},
		{name: "globstar suffix zero segments", pattern: "legacy/**", path: "legacy", want: true},
		{name: "no match", pattern: "**/prod/**", path: "platform/stage/eu-central-1/app", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := MatchGlob(tt.pattern, tt.path)
			if err != nil {
				t.Fatalf("MatchGlob() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("MatchGlob(%q, %q) = %v, want %v", tt.pattern, tt.path, got, tt.want)
			}
		})
	}
}

func TestValidateGlobRejectsMalformedPatterns(t *testing.T) {
	t.Parallel()

	tests := []string{
		"platform/[bad/**",
		"platform/**bad/vpc",
	}

	for _, pattern := range tests {
		t.Run(pattern, func(t *testing.T) {
			t.Parallel()

			if err := ValidateGlob(pattern); err == nil {
				t.Fatalf("ValidateGlob(%q) error = nil, want error", pattern)
			}
		})
	}
}
