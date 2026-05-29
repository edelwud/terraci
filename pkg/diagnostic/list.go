package diagnostic

import (
	"sort"
)

// List is an immutable, deterministic collection of diagnostics.
type List struct {
	diagnostics []Diagnostic
}

// NewList constructs a sorted and deduplicated diagnostic list.
func NewList(diags ...Diagnostic) List {
	if len(diags) == 0 {
		return List{}
	}
	byKey := make(map[string]Diagnostic, len(diags))
	for _, diag := range diags {
		if !diag.Valid() {
			continue
		}
		byKey[diag.key()] = diag
	}
	if len(byKey) == 0 {
		return List{}
	}
	out := make([]Diagnostic, 0, len(byKey))
	for _, diag := range byKey {
		out = append(out, diag)
	}
	sort.Slice(out, func(i, j int) bool {
		left, right := out[i], out[j]
		if left.severity.rank() != right.severity.rank() {
			return left.severity.rank() < right.severity.rank()
		}
		return left.key() < right.key()
	})
	return List{diagnostics: out}
}

// FromWarnings converts legacy warning strings into warning diagnostics.
func FromWarnings(warnings []string, opts ...Option) List {
	diags := make([]Diagnostic, 0, len(warnings))
	for _, warning := range warnings {
		diags = append(diags, Warning(warning, opts...))
	}
	return NewList(diags...)
}

// Len returns the number of diagnostics.
func (l List) Len() int { return len(l.diagnostics) }

// Empty reports whether the list has no diagnostics.
func (l List) Empty() bool { return len(l.diagnostics) == 0 }

// All returns defensive diagnostic copies in deterministic order.
func (l List) All() []Diagnostic {
	if len(l.diagnostics) == 0 {
		return nil
	}
	return append([]Diagnostic(nil), l.diagnostics...)
}

// Messages returns human-facing messages in deterministic order.
func (l List) Messages() []string {
	if len(l.diagnostics) == 0 {
		return nil
	}
	out := make([]string, 0, len(l.diagnostics))
	for _, diag := range l.diagnostics {
		out = append(out, diag.Message())
	}
	return out
}

// Append returns a new list containing the receiver and supplied diagnostics.
func (l List) Append(diags ...Diagnostic) List {
	merged := l.All()
	merged = append(merged, diags...)
	return NewList(merged...)
}

// Merge returns a new list containing diagnostics from both lists.
func (l List) Merge(other List) List {
	merged := l.All()
	merged = append(merged, other.All()...)
	return NewList(merged...)
}

// Filter returns diagnostics matching the supplied severity.
func (l List) Filter(severity Severity) []Diagnostic {
	if len(l.diagnostics) == 0 {
		return nil
	}
	out := make([]Diagnostic, 0)
	for _, diag := range l.diagnostics {
		if diag.Severity() == severity {
			out = append(out, diag)
		}
	}
	return out
}
