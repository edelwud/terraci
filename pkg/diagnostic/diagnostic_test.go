package diagnostic

import (
	"errors"
	"reflect"
	"testing"
)

func TestNewValidatesSeverityAndMessage(t *testing.T) {
	t.Parallel()

	if _, err := New(Options{Severity: Severity("bad"), Message: "broken"}); err == nil {
		t.Fatal("New() error = nil, want invalid severity")
	}
	if _, err := New(Options{Severity: SeverityWarning}); err == nil {
		t.Fatal("New() error = nil, want missing message")
	}
}

func TestDiagnosticWrapsCauseAndRendersContext(t *testing.T) {
	t.Parallel()

	cause := errors.New("network failed")
	diag, err := New(Options{
		Severity: SeverityWarning,
		Message:  "stale report skipped",
		Source:   "summary",
		Module:   "svc/prod/eu/vpc",
		Hint:     "rerun producer",
		Err:      cause,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if !errors.Is(diag.Cause(), cause) {
		t.Fatalf("Cause() = %v, want wrapped cause", diag.Cause())
	}
	if got, want := diag.String(), "summary svc/prod/eu/vpc: stale report skipped (rerun producer)"; got != want {
		t.Fatalf("String() = %q, want %q", got, want)
	}
}

func TestListSortsDedupesAndDefensivelyCopies(t *testing.T) {
	t.Parallel()

	list := NewList(
		Info("later"),
		Warning("same"),
		Error("first"),
		Warning("same"),
		Warning("other", WithSource("b")),
		Warning("other", WithSource("a")),
	)

	gotMessages := list.Messages()
	wantMessages := []string{"first", "other", "other", "same", "later"}
	if !reflect.DeepEqual(gotMessages, wantMessages) {
		t.Fatalf("Messages() = %#v, want %#v", gotMessages, wantMessages)
	}

	all := list.All()
	all[0] = Info("mutated")
	if got := list.All()[0].Message(); got != "first" {
		t.Fatalf("list leaked mutation, first message = %q", got)
	}
}

func TestFromWarnings(t *testing.T) {
	t.Parallel()

	list := FromWarnings([]string{"b", "a", "a"}, WithSource("ci"))
	if got, want := list.Len(), 2; got != want {
		t.Fatalf("Len() = %d, want %d", got, want)
	}
	for _, diag := range list.All() {
		if diag.Severity() != SeverityWarning || diag.Source() != "ci" {
			t.Fatalf("diag = %#v, want warning with ci source", diag)
		}
	}
}
