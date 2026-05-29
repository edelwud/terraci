package execution

import "fmt"

// ExecutionError wraps a failed job execution while preserving the original
// cause for errors.Is/errors.As.
//
//nolint:revive // Explicit typed error name is part of the public local-exec contract.
type ExecutionError struct {
	JobName string
	Err     error
}

func (e *ExecutionError) Error() string {
	if e == nil {
		return "execution error"
	}
	if e.JobName == "" {
		return fmt.Sprintf("execution failed: %v", e.Err)
	}
	return fmt.Sprintf("execution failed in job %s: %v", e.JobName, e.Err)
}

func (e *ExecutionError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}
