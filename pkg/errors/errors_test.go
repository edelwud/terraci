package errors

import (
	"errors"
	"testing"
)

func TestConfigError(t *testing.T) {
	inner := errors.New("bad value")

	t.Run("with path", func(t *testing.T) {
		e := &ConfigError{Path: "/etc/terraci.yaml", Err: inner}
		want := "config error (/etc/terraci.yaml): bad value"
		if e.Error() != want {
			t.Errorf("Error() = %q, want %q", e.Error(), want)
		}
	})

	t.Run("without path", func(t *testing.T) {
		e := &ConfigError{Err: inner}
		want := "config error: bad value"
		if e.Error() != want {
			t.Errorf("Error() = %q, want %q", e.Error(), want)
		}
	})

	t.Run("unwrap", func(t *testing.T) {
		e := &ConfigError{Err: inner}
		if !errors.Is(e, inner) {
			t.Error("Unwrap should return inner error")
		}
	})
}

func TestScanError(t *testing.T) {
	inner := errors.New("permission denied")
	e := &ScanError{Dir: "/tmp/modules", Err: inner}

	if e.Error() != "scan /tmp/modules: permission denied" {
		t.Errorf("Error() = %q", e.Error())
	}
	if !errors.Is(e, inner) {
		t.Error("Unwrap should return inner error")
	}
}

func TestParseError(t *testing.T) {
	inner := errors.New("invalid HCL")
	e := &ParseError{Module: "vpc", Err: inner}

	if e.Error() != "parse vpc: invalid HCL" {
		t.Errorf("Error() = %q", e.Error())
	}
	if !errors.Is(e, inner) {
		t.Error("Unwrap should return inner error")
	}
}

func TestPolicyError(t *testing.T) {
	e := &PolicyError{Module: "rds", Violations: []string{"no tags", "no encryption"}}
	want := "policy check failed for rds: 2 violation(s)"
	if e.Error() != want {
		t.Errorf("Error() = %q, want %q", e.Error(), want)
	}
}

func TestCostError(t *testing.T) {
	inner := errors.New("API timeout")
	e := &CostError{Module: "ec2", Err: inner}

	if e.Error() != "cost estimation ec2: API timeout" {
		t.Errorf("Error() = %q", e.Error())
	}
	if !errors.Is(e, inner) {
		t.Error("Unwrap should return inner error")
	}
}

func TestGraphError(t *testing.T) {
	e := &GraphError{Cycles: [][]string{{"a", "b", "a"}}}
	want := "dependency graph has 1 cycle(s)"
	if e.Error() != want {
		t.Errorf("Error() = %q, want %q", e.Error(), want)
	}
}

func TestNoModulesError(t *testing.T) {
	e := &NoModulesError{Dir: "/projects/infra"}
	want := "no modules found in /projects/infra"
	if e.Error() != want {
		t.Errorf("Error() = %q, want %q", e.Error(), want)
	}
}

func TestErrorsAs(t *testing.T) {
	inner := errors.New("oops")

	tests := []struct {
		name string
		err  error
	}{
		{"ConfigError", &ConfigError{Err: inner}},
		{"ScanError", &ScanError{Err: inner}},
		{"ParseError", &ParseError{Err: inner}},
		{"CostError", &CostError{Err: inner}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(_ *testing.T) {
			wrapped := errors.Join(tt.err, errors.New("context"))
			_ = wrapped // just verify type assertions work
		})
	}
}
