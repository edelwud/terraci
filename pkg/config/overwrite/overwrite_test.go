package overwrite

import (
	"errors"
	"strings"
	"testing"
)

type testOverwrite struct {
	Key    string
	Match  string
	Label  string
	Values []string
}

type testEffective struct {
	Label  string
	Values []string
}

func TestApplyMatching_ByKeyAppliesAllMatchesInOrder(t *testing.T) {
	t.Parallel()

	effective := testEffective{Label: "base", Values: []string{"base"}}
	overwrites := []testOverwrite{
		{Key: "plan", Label: "first", Values: []string{"one"}},
		{Key: "apply", Label: "skip", Values: []string{"skip"}},
		{Key: "plan", Label: "second", Values: []string{"two"}},
	}

	err := ApplyMatching(
		&effective,
		"plan",
		overwrites,
		ByKey(func(ow *testOverwrite) string { return ow.Key }),
		func(e *testEffective, ow *testOverwrite) {
			if ow.Label != "" {
				e.Label = ow.Label
			}
			e.Values = append(e.Values, ow.Values...)
		},
	)
	if err != nil {
		t.Fatalf("ApplyMatching() error = %v", err)
	}

	if effective.Label != "second" {
		t.Fatalf("Label = %q, want second", effective.Label)
	}
	if got := strings.Join(effective.Values, ","); got != "base,one,two" {
		t.Fatalf("Values = %q, want base,one,two", got)
	}
}

func TestResolveReturnsEffectiveCopy(t *testing.T) {
	t.Parallel()

	base := testEffective{Label: "base"}
	got, err := Resolve(
		base,
		"apply",
		[]testOverwrite{{Key: "apply", Label: "override"}},
		ByKey(func(ow *testOverwrite) string { return ow.Key }),
		func(e *testEffective, ow *testOverwrite) {
			e.Label = ow.Label
		},
	)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if got.Label != "override" {
		t.Fatalf("resolved Label = %q, want override", got.Label)
	}
	if base.Label != "base" {
		t.Fatalf("base Label = %q, want base", base.Label)
	}
}

func TestApplyMatchingWrapsMatcherErrorsWithIndex(t *testing.T) {
	t.Parallel()

	effective := testEffective{}
	err := ApplyMatching(
		&effective,
		"target",
		[]testOverwrite{{Key: "ok"}, {Key: "bad"}},
		func(ow *testOverwrite, _ string) (bool, error) {
			if ow.Key == "bad" {
				return false, errors.New("bad pattern")
			}
			return false, nil
		},
		func(_ *testEffective, _ *testOverwrite) {},
	)
	if err == nil {
		t.Fatal("ApplyMatching() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "overwrites[1]") || !strings.Contains(err.Error(), "bad pattern") {
		t.Fatalf("error = %q, want overwrite index and cause", err)
	}
}

func TestMatchPathGlob(t *testing.T) {
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
		{name: "globstar prefix", pattern: "**/sandbox/**", path: "platform/sandbox/eu-central-1/app", want: true},
		{name: "globstar suffix zero segments", pattern: "legacy/**", path: "legacy", want: true},
		{name: "no match", pattern: "**/prod/**", path: "platform/stage/eu-central-1/app", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := MatchPathGlob(tt.pattern, tt.path)
			if err != nil {
				t.Fatalf("MatchPathGlob() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("MatchPathGlob(%q, %q) = %v, want %v", tt.pattern, tt.path, got, tt.want)
			}
		})
	}
}

func TestValidatePathGlobRejectsMalformedPatterns(t *testing.T) {
	t.Parallel()

	tests := []string{
		"platform/[bad/**",
		"platform/**bad/vpc",
	}

	for _, pattern := range tests {
		t.Run(pattern, func(t *testing.T) {
			t.Parallel()

			if err := ValidatePathGlob(pattern); err == nil {
				t.Fatalf("ValidatePathGlob(%q) error = nil, want error", pattern)
			}
		})
	}
}

func TestByPathGlobUsesExtractedPattern(t *testing.T) {
	t.Parallel()

	effective := testEffective{}
	err := ApplyMatching(
		&effective,
		"platform/prod/eu-central-1/app",
		[]testOverwrite{{Match: "**/stage/**", Label: "stage"}, {Match: "**/prod/**", Label: "prod"}},
		ByPathGlob(func(ow *testOverwrite) string { return ow.Match }),
		func(e *testEffective, ow *testOverwrite) {
			e.Label = ow.Label
		},
	)
	if err != nil {
		t.Fatalf("ApplyMatching() error = %v", err)
	}
	if effective.Label != "prod" {
		t.Fatalf("Label = %q, want prod", effective.Label)
	}
}
