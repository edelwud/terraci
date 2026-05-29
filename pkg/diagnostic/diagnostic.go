// Package diagnostic provides small immutable diagnostic value objects for
// non-fatal messages that need stable severity and optional error context.
package diagnostic

import (
	"errors"
	"fmt"
	"strings"
)

// Severity classifies a diagnostic for rendering and filtering.
type Severity string

const (
	SeverityInfo    Severity = "info"
	SeverityWarning Severity = "warning"
	SeverityError   Severity = "error"
)

// Valid reports whether the severity is one of the supported values.
func (s Severity) Valid() bool {
	switch s {
	case SeverityInfo, SeverityWarning, SeverityError:
		return true
	default:
		return false
	}
}

func (s Severity) String() string {
	if s == "" {
		return string(SeverityInfo)
	}
	return string(s)
}

func (s Severity) rank() int {
	switch s {
	case SeverityError:
		return 0
	case SeverityWarning:
		return 1
	case SeverityInfo:
		return 2
	default:
		return 3
	}
}

// Options describes one diagnostic.
type Options struct {
	Severity Severity
	Message  string
	Source   string
	Module   string
	Hint     string
	Err      error
}

// Option customizes a diagnostic constructor.
type Option func(*Options)

// WithSource attaches a producer/subsystem label.
func WithSource(source string) Option {
	return func(opts *Options) {
		opts.Source = source
	}
}

// WithModule attaches a TerraCi module identifier or path.
func WithModule(module string) Option {
	return func(opts *Options) {
		opts.Module = module
	}
}

// WithHint attaches an actionable user hint.
func WithHint(hint string) Option {
	return func(opts *Options) {
		opts.Hint = hint
	}
}

// WithCause attaches a wrapped cause.
func WithCause(err error) Option {
	return func(opts *Options) {
		opts.Err = err
	}
}

// Diagnostic is one immutable diagnostic message.
type Diagnostic struct {
	severity Severity
	message  string
	source   string
	module   string
	hint     string
	err      error
}

// New validates and constructs one diagnostic.
func New(opts Options) (Diagnostic, error) {
	severity := opts.Severity
	if severity == "" {
		severity = SeverityInfo
	}
	if !severity.Valid() {
		return Diagnostic{}, fmt.Errorf("invalid diagnostic severity %q", severity)
	}

	message := strings.TrimSpace(opts.Message)
	if message == "" && opts.Err != nil {
		message = opts.Err.Error()
	}
	if message == "" {
		return Diagnostic{}, errors.New("diagnostic message is required")
	}

	return Diagnostic{
		severity: severity,
		message:  message,
		source:   strings.TrimSpace(opts.Source),
		module:   strings.TrimSpace(opts.Module),
		hint:     strings.TrimSpace(opts.Hint),
		err:      opts.Err,
	}, nil
}

// Info constructs an informational diagnostic.
func Info(message string, opts ...Option) Diagnostic {
	return mustBuild(SeverityInfo, message, opts...)
}

// Warning constructs a warning diagnostic.
func Warning(message string, opts ...Option) Diagnostic {
	return mustBuild(SeverityWarning, message, opts...)
}

// Error constructs an error diagnostic.
func Error(message string, opts ...Option) Diagnostic {
	return mustBuild(SeverityError, message, opts...)
}

func mustBuild(severity Severity, message string, opts ...Option) Diagnostic {
	options := Options{Severity: severity, Message: message}
	for _, opt := range opts {
		if opt != nil {
			opt(&options)
		}
	}
	diag, err := New(options)
	if err != nil {
		if options.Err != nil {
			return Diagnostic{severity: severity, message: options.Err.Error(), err: options.Err}
		}
		return Diagnostic{severity: severity, message: severity.String()}
	}
	return diag
}

// Valid reports whether this diagnostic has a supported severity and message.
func (d Diagnostic) Valid() bool {
	return d.severity.Valid() && d.message != ""
}

// Severity returns the diagnostic severity.
func (d Diagnostic) Severity() Severity { return d.severity }

// Message returns the human-facing diagnostic message without context prefixing.
func (d Diagnostic) Message() string { return d.message }

// Source returns the optional producer/subsystem label.
func (d Diagnostic) Source() string { return d.source }

// Module returns the optional module identifier or path.
func (d Diagnostic) Module() string { return d.module }

// Hint returns the optional actionable hint.
func (d Diagnostic) Hint() string { return d.hint }

// Cause returns the wrapped diagnostic cause, if any.
func (d Diagnostic) Cause() error { return d.err }

// String returns a deterministic context-rich diagnostic string.
func (d Diagnostic) String() string {
	if !d.Valid() {
		return ""
	}
	parts := make([]string, 0, 3)
	if d.source != "" {
		parts = append(parts, d.source)
	}
	if d.module != "" {
		parts = append(parts, d.module)
	}
	prefix := strings.Join(parts, " ")
	out := d.message
	if prefix != "" {
		out = prefix + ": " + out
	}
	if d.hint != "" {
		out += " (" + d.hint + ")"
	}
	return out
}

func (d Diagnostic) key() string {
	return strings.Join([]string{
		d.severity.String(),
		d.message,
		d.source,
		d.module,
		d.hint,
	}, "\x00")
}
