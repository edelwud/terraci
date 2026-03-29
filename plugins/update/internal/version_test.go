package updateengine

import (
	"testing"
)

func TestVersion_String(t *testing.T) {
	tests := []struct {
		v    Version
		want string
	}{
		{Version{1, 2, 3, ""}, "1.2.3"},
		{Version{1, 2, 3, "beta"}, "1.2.3-beta"},
		{Version{0, 0, 0, ""}, "0.0.0"},
	}

	for _, tt := range tests {
		if got := tt.v.String(); got != tt.want {
			t.Errorf("%v.String() = %q, want %q", tt.v, got, tt.want)
		}
	}
}

func TestVersion_IsZero(t *testing.T) {
	tests := []struct {
		v    Version
		want bool
	}{
		{Version{0, 0, 0, ""}, true},
		{Version{1, 0, 0, ""}, false},
		{Version{0, 0, 1, ""}, false},
		{Version{0, 0, 0, "beta"}, false},
	}

	for _, tt := range tests {
		if got := tt.v.IsZero(); got != tt.want {
			t.Errorf("%v.IsZero() = %v, want %v", tt.v, got, tt.want)
		}
	}
}

func TestParseVersion(t *testing.T) {
	tests := []struct {
		input   string
		want    Version
		wantErr bool
	}{
		{"1.2.3", Version{1, 2, 3, ""}, false},
		{"0.1.0", Version{0, 1, 0, ""}, false},
		{"5.67.0", Version{5, 67, 0, ""}, false},
		{"v1.2.3", Version{1, 2, 3, ""}, false},
		{"1.2.3-beta", Version{1, 2, 3, "beta"}, false},
		{"1.0", Version{1, 0, 0, ""}, false},
		{"5", Version{5, 0, 0, ""}, false},
		{"", Version{}, true},
		{"abc", Version{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseVersion(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseVersion(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("ParseVersion(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestVersionCompare(t *testing.T) {
	tests := []struct {
		a, b Version
		want int
	}{
		{Version{1, 0, 0, ""}, Version{2, 0, 0, ""}, -1},
		{Version{1, 0, 0, ""}, Version{1, 0, 0, ""}, 0},
		{Version{2, 0, 0, ""}, Version{1, 0, 0, ""}, 1},
		{Version{1, 2, 0, ""}, Version{1, 1, 0, ""}, 1},
		{Version{1, 0, 1, ""}, Version{1, 0, 0, ""}, 1},
		{Version{1, 0, 0, "beta"}, Version{1, 0, 0, ""}, -1},
		{Version{1, 0, 0, ""}, Version{1, 0, 0, "beta"}, 1},
	}

	for _, tt := range tests {
		got := tt.a.Compare(tt.b)
		if got != tt.want {
			t.Errorf("%v.Compare(%v) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestParseConstraints(t *testing.T) {
	tests := []struct {
		input string
		count int
	}{
		{"~> 5.0", 1},
		{">= 1.0, < 2.0", 2},
		{"= 1.2.3", 1},
		{"~> 5.0.1", 1},
		{">= 3.0", 1},
		{"", 0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseConstraints(tt.input)
			if err != nil {
				t.Fatalf("ParseConstraints(%q) error: %v", tt.input, err)
			}
			if len(got) != tt.count {
				t.Errorf("ParseConstraints(%q) returned %d constraints, want %d", tt.input, len(got), tt.count)
			}
		})
	}
}

func TestPessimisticConstraint(t *testing.T) {
	tests := []struct {
		constraint string
		version    string
		want       bool
	}{
		// ~> 5.0 means >= 5.0, < 6.0
		{"~> 5.0", "5.0.0", true},
		{"~> 5.0", "5.5.0", true},
		{"~> 5.0", "5.99.0", true},
		{"~> 5.0", "6.0.0", false},
		{"~> 5.0", "4.9.0", false},
		// ~> 5.0.1 means >= 5.0.1, < 5.1.0
		{"~> 5.0.1", "5.0.1", true},
		{"~> 5.0.1", "5.0.9", true},
		{"~> 5.0.1", "5.1.0", false},
		{"~> 5.0.1", "5.0.0", false},
	}

	for _, tt := range tests {
		t.Run(tt.constraint+"_"+tt.version, func(t *testing.T) {
			constraints, err := ParseConstraints(tt.constraint)
			if err != nil {
				t.Fatal(err)
			}
			v, err := ParseVersion(tt.version)
			if err != nil {
				t.Fatal(err)
			}
			got := SatisfiesAll(v, constraints)
			if got != tt.want {
				t.Errorf("SatisfiesAll(%s, %s) = %v, want %v", tt.version, tt.constraint, got, tt.want)
			}
		})
	}
}

func TestLatestByBump(t *testing.T) {
	versions := []Version{
		{5, 0, 0, ""}, {5, 0, 1, ""}, {5, 1, 0, ""}, {5, 2, 0, ""},
		{6, 0, 0, ""}, {6, 1, 0, ""},
	}
	current := Version{5, 0, 0, ""}

	// Patch: same major.minor only.
	got, ok := LatestByBump(current, versions, BumpPatch)
	if !ok || got != (Version{5, 0, 1, ""}) {
		t.Errorf("patch bump = %v (%v), want 5.0.1", got, ok)
	}

	// Minor: same major.
	got, ok = LatestByBump(current, versions, BumpMinor)
	if !ok || got != (Version{5, 2, 0, ""}) {
		t.Errorf("minor bump = %v (%v), want 5.2.0", got, ok)
	}

	// Major: absolute latest.
	got, ok = LatestByBump(current, versions, BumpMajor)
	if !ok || got != (Version{6, 1, 0, ""}) {
		t.Errorf("major bump = %v (%v), want 6.1.0", got, ok)
	}
}

func TestSatisfiesAll_EmptyConstraints(t *testing.T) {
	v := Version{5, 0, 0, ""}
	if !SatisfiesAll(v, nil) {
		t.Error("SatisfiesAll with nil constraints should return true")
	}
	if !SatisfiesAll(v, []Constraint{}) {
		t.Error("SatisfiesAll with empty constraints should return true")
	}
}

func TestConstraint_Satisfies_AllOperators(t *testing.T) {
	v100 := Version{1, 0, 0, ""}
	v200 := Version{2, 0, 0, ""}

	tests := []struct {
		name string
		c    Constraint
		v    Version
		want bool
	}{
		{"equal match", Constraint{Op: OpEqual, Version: v100}, v100, true},
		{"equal no match", Constraint{Op: OpEqual, Version: v100}, v200, false},
		{"not equal match", Constraint{Op: OpNotEqual, Version: v100}, v200, true},
		{"not equal no match", Constraint{Op: OpNotEqual, Version: v100}, v100, false},
		{"greater match", Constraint{Op: OpGreater, Version: v100}, v200, true},
		{"greater no match", Constraint{Op: OpGreater, Version: v200}, v100, false},
		{"greater equal match", Constraint{Op: OpGreaterEqual, Version: v100}, v100, true},
		{"greater equal no match", Constraint{Op: OpGreaterEqual, Version: v200}, v100, false},
		{"less match", Constraint{Op: OpLess, Version: v200}, v100, true},
		{"less no match", Constraint{Op: OpLess, Version: v100}, v200, false},
		{"less equal match", Constraint{Op: OpLessEqual, Version: v100}, v100, true},
		{"less equal no match", Constraint{Op: OpLessEqual, Version: v100}, v200, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.c.Satisfies(tt.v); got != tt.want {
				t.Errorf("Satisfies() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLatestAllowed(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		versions := []Version{{1, 0, 0, ""}, {2, 0, 0, ""}, {3, 0, 0, ""}}
		constraints := []Constraint{{Op: OpPessimistic, Version: Version{2, 0, 0, ""}, Parts: 2}}
		got, ok := LatestAllowed(versions, constraints)
		if !ok {
			t.Fatal("expected to find a version")
		}
		if got != (Version{2, 0, 0, ""}) {
			t.Errorf("LatestAllowed = %v, want 2.0.0", got)
		}
	})

	t.Run("not_found", func(t *testing.T) {
		versions := []Version{{1, 0, 0, ""}}
		constraints := []Constraint{{Op: OpGreater, Version: Version{5, 0, 0, ""}}}
		_, ok := LatestAllowed(versions, constraints)
		if ok {
			t.Error("expected not found")
		}
	})

	t.Run("skips_prereleases", func(t *testing.T) {
		versions := []Version{{1, 0, 0, ""}, {2, 0, 0, "beta"}}
		constraints := []Constraint{{Op: OpGreaterEqual, Version: Version{1, 0, 0, ""}}}
		got, ok := LatestAllowed(versions, constraints)
		if !ok {
			t.Fatal("expected to find a version")
		}
		if got != (Version{1, 0, 0, ""}) {
			t.Errorf("LatestAllowed = %v, want 1.0.0", got)
		}
	})
}

func TestParseSingleConstraint_AllOperators(t *testing.T) {
	tests := []struct {
		input  string
		wantOp ConstraintOp
	}{
		{"~> 5.0", OpPessimistic},
		{">= 1.0", OpGreaterEqual},
		{"<= 1.0", OpLessEqual},
		{"!= 1.0", OpNotEqual},
		{"> 1.0", OpGreater},
		{"< 1.0", OpLess},
		{"= 1.0", OpEqual},
		{"1.0", OpEqual},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			c, err := parseSingleConstraint(tt.input)
			if err != nil {
				t.Fatalf("parseSingleConstraint(%q) error = %v", tt.input, err)
			}
			if c.Op != tt.wantOp {
				t.Errorf("Op = %v, want %v", c.Op, tt.wantOp)
			}
		})
	}
}

func TestParseConstraints_EmptyParts(t *testing.T) {
	got, err := ParseConstraints("~> 5.0, , >= 1.0")
	if err != nil {
		t.Fatalf("ParseConstraints error = %v", err)
	}
	if len(got) != 2 {
		t.Errorf("len = %d, want 2 (empty middle skipped)", len(got))
	}
}

func TestLatestByBump_NoMatch(t *testing.T) {
	// Current is already the latest — no bump available.
	versions := []Version{{5, 0, 0, ""}}
	current := Version{5, 0, 0, ""}

	_, ok := LatestByBump(current, versions, BumpPatch)
	if ok {
		t.Error("expected no match when current is latest")
	}
}

func TestLatestByBump_SkipsPrereleases(t *testing.T) {
	versions := []Version{
		{5, 0, 0, ""}, {5, 0, 1, "rc1"}, {5, 0, 2, ""},
	}
	current := Version{5, 0, 0, ""}

	got, ok := LatestByBump(current, versions, BumpPatch)
	if !ok {
		t.Fatal("expected a match")
	}
	if got != (Version{5, 0, 2, ""}) {
		t.Errorf("got %v, want 5.0.2 (skipping prerelease)", got)
	}
}

func TestBumpConstraint_Complex(t *testing.T) {
	// Complex constraint (multiple constraints) falls back to pessimistic
	got := BumpConstraint(">= 1.0, < 2.0", Version{1, 5, 0, ""})
	if got != "~> 1.5" {
		t.Errorf("BumpConstraint(complex) = %q, want '~> 1.5'", got)
	}
}

func TestBumpConstraint_ParseError(t *testing.T) {
	// Unparseable constraint returns original
	got := BumpConstraint("???", Version{1, 0, 0, ""})
	if got != "???" {
		t.Errorf("BumpConstraint(invalid) = %q, want '???'", got)
	}
}

func TestBumpConstraint_LessOperator(t *testing.T) {
	// < operator is not handled by the switch — falls through to default pessimistic
	got := BumpConstraint("< 5.0", Version{4, 5, 0, ""})
	if got != "~> 4.5" {
		t.Errorf("BumpConstraint(< 5.0) = %q, want '~> 4.5'", got)
	}
}

func TestBumpConstraint_NotEqualOperator(t *testing.T) {
	// != operator is not handled by the switch
	got := BumpConstraint("!= 3.0", Version{4, 0, 0, ""})
	if got != "~> 4.0" {
		t.Errorf("BumpConstraint(!= 3.0) = %q, want '~> 4.0'", got)
	}
}

func TestBumpConstraint_Empty(t *testing.T) {
	got := BumpConstraint("", Version{1, 0, 0, ""})
	if got != "" {
		t.Errorf("BumpConstraint(empty) = %q, want empty", got)
	}
}

func TestConstraint_Satisfies_UnknownOp(t *testing.T) {
	c := Constraint{Op: ConstraintOp(99), Version: Version{1, 0, 0, ""}}
	if c.Satisfies(Version{1, 0, 0, ""}) {
		t.Error("unknown op should return false")
	}
}

func TestParseVersion_TooManyParts(t *testing.T) {
	_, err := ParseVersion("1.2.3.4")
	if err == nil {
		t.Error("expected error for > 3 parts")
	}
}

func TestBumpConstraint(t *testing.T) {
	tests := []struct {
		original   string
		newVersion Version
		want       string
	}{
		{"~> 5.0", Version{5, 3, 0, ""}, "~> 5.3"},
		{"~> 5.0", Version{6, 0, 0, ""}, "~> 6.0"},
		{"~> 5.0.1", Version{5, 0, 3, ""}, "~> 5.0.3"},
		{"= 1.2.3", Version{1, 2, 5, ""}, "= 1.2.5"},
		{">= 1.0", Version{2, 0, 0, ""}, ">= 2.0"},
		{">= 1.0.0", Version{2, 0, 0, ""}, ">= 2.0.0"},
	}

	for _, tt := range tests {
		t.Run(tt.original, func(t *testing.T) {
			got := BumpConstraint(tt.original, tt.newVersion)
			if got != tt.want {
				t.Errorf("BumpConstraint(%q, %v) = %q, want %q", tt.original, tt.newVersion, got, tt.want)
			}
		})
	}
}
