// Package errors provides structured error types whose call sites need to
// match on type via errors.As (e.g. CLI exit-code mapping, retry policies).
//
// Lower-level wrap-and-propagate paths (config load, HCL parse) use plain
// fmt.Errorf("...: %w", err) — wrapping that nobody dispatches on doesn't
// earn its own type.
package errors

import "fmt"

// ScanError indicates a module discovery failure; matched by tests and CLI
// callers via errors.As to distinguish "could not scan workdir" from other
// workflow errors.
type ScanError struct {
	Dir string
	Err error
}

func (e *ScanError) Error() string {
	return fmt.Sprintf("scan %s: %s", e.Dir, e.Err)
}

func (e *ScanError) Unwrap() error { return e.Err }

// NoModulesError indicates that module discovery succeeded but found no
// modules. Distinct from ScanError so the CLI can emit a friendly hint
// instead of a generic failure message.
type NoModulesError struct {
	Dir string
}

func (e *NoModulesError) Error() string {
	return "no modules found in " + e.Dir
}
