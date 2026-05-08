package workspacepath

import "testing"

func TestJoinPreservesInvalidSegmentsForValidation(t *testing.T) {
	t.Parallel()

	got := Join("svc/../vpc", "plan.json")
	if got != "svc/../vpc/plan.json" {
		t.Fatalf("Join() = %q, want parent segment preserved", got)
	}
	if err := Validate(got); err == nil {
		t.Fatal("Validate() error = nil, want parent segment rejected")
	}
}

func TestValidateOptionalAllowsEmptyOnly(t *testing.T) {
	t.Parallel()

	if err := ValidateOptional(""); err != nil {
		t.Fatalf("ValidateOptional(\"\") error = %v", err)
	}
	if err := Validate(""); err == nil {
		t.Fatal("Validate(\"\") error = nil, want error")
	}
}

func TestValidateRejectsWorkspaceEscapes(t *testing.T) {
	t.Parallel()

	for _, value := range []string{
		"/tmp/plan.json",
		"../plan.json",
		"svc/../plan.json",
		`C:\tmp\plan.json`,
		`svc/C:\tmp/plan.json`,
	} {
		if err := Validate(value); err == nil {
			t.Fatalf("Validate(%q) error = nil, want error", value)
		}
	}
}
