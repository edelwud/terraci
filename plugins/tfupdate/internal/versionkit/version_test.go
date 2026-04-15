package versionkit

import "testing"

func TestParseVersion(t *testing.T) {
	got, err := ParseVersion("v1.2.3-beta")
	if err != nil {
		t.Fatalf("ParseVersion() error = %v", err)
	}
	want := Version{Major: 1, Minor: 2, Patch: 3, Prerelease: "beta"}
	if got != want {
		t.Fatalf("ParseVersion() = %#v, want %#v", got, want)
	}
}

func TestPessimisticConstraint(t *testing.T) {
	tests := []struct {
		name       string
		constraint string
		version    string
		want       bool
	}{
		{name: "two part allows same major", constraint: "~> 5.0", version: "5.99.0", want: true},
		{name: "two part blocks next major", constraint: "~> 5.0", version: "6.0.0", want: false},
		{name: "three part allows same minor", constraint: "~> 5.0.1", version: "5.0.9", want: true},
		{name: "three part blocks next minor", constraint: "~> 5.0.1", version: "5.1.0", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			constraints, err := ParseConstraints(tt.constraint)
			if err != nil {
				t.Fatalf("ParseConstraints() error = %v", err)
			}
			version, err := ParseVersion(tt.version)
			if err != nil {
				t.Fatalf("ParseVersion() error = %v", err)
			}
			if got := SatisfiesAll(version, constraints); got != tt.want {
				t.Fatalf("SatisfiesAll() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBumpConstraint(t *testing.T) {
	got := BumpConstraint("~> 5.0", Version{Major: 5, Minor: 3})
	if got != "~> 5.3" {
		t.Fatalf("BumpConstraint() = %q", got)
	}
}

func TestMergeConstraints(t *testing.T) {
	got := MergeConstraints("~> 5.0", []string{">= 5.0.0, < 6.0.0"})
	want := ">= 5.0.0, ~> 5.0, < 6.0.0"
	if got != want {
		t.Fatalf("MergeConstraints() = %q, want %q", got, want)
	}
}
