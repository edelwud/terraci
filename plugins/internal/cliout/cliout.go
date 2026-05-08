// Package cliout contains shared CLI output helpers used by built-in plugins.
package cliout

import (
	"encoding/json"
	"io"
)

// WriteJSON writes v to w as pretty-printed JSON with 2-space indentation.
// Used by plugin output renderers (cost/policy/tfupdate) to emit structured
// command output when --output=json is set.
func WriteJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
