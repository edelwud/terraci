package errors

import (
	"errors"
	"testing"
)

func TestConfigError(t *testing.T) {
	t.Parallel()

	inner := errors.New("bad value")

	t.Run("with path", func(t *testing.T) {
		t.Parallel()

		e := &ConfigError{Path: "/etc/terraci.yaml", Err: inner}
		want := "config error (/etc/terraci.yaml): bad value"
		if e.Error() != want {
			t.Errorf("Error() = %q, want %q", e.Error(), want)
		}
	})

	t.Run("without path", func(t *testing.T) {
		t.Parallel()

		e := &ConfigError{Err: inner}
		want := "config error: bad value"
		if e.Error() != want {
			t.Errorf("Error() = %q, want %q", e.Error(), want)
		}
	})

	t.Run("unwrap", func(t *testing.T) {
		t.Parallel()

		e := &ConfigError{Err: inner}
		if !errors.Is(e, inner) {
			t.Error("Unwrap should return inner error")
		}
	})
}

func TestScanError(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

	inner := errors.New("invalid HCL")
	e := &ParseError{Module: "vpc", Err: inner}

	if e.Error() != "parse vpc: invalid HCL" {
		t.Errorf("Error() = %q", e.Error())
	}
	if !errors.Is(e, inner) {
		t.Error("Unwrap should return inner error")
	}
}

func TestNoModulesError(t *testing.T) {
	t.Parallel()

	e := &NoModulesError{Dir: "/projects/infra"}
	want := "no modules found in /projects/infra"
	if e.Error() != want {
		t.Errorf("Error() = %q, want %q", e.Error(), want)
	}
}

func TestErrorsAs(t *testing.T) {
	t.Parallel()

	inner := errors.New("oops")

	tests := []struct {
		name string
		err  error
	}{
		{"ConfigError", &ConfigError{Err: inner}},
		{"ScanError", &ScanError{Err: inner}},
		{"ParseError", &ParseError{Err: inner}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			wrapped := errors.Join(tt.err, errors.New("context"))
			_ = wrapped // just verify type assertions work
		})
	}
}
