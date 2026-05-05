package errors

import (
	"errors"
	"testing"
)

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

	wrapped := errors.Join(&ScanError{Err: inner}, errors.New("context"))
	var scanErr *ScanError
	if !errors.As(wrapped, &scanErr) {
		t.Fatal("errors.As should find ScanError in joined chain")
	}
}
