// Package cliout contains shared CLI output helpers for plugin commands.
package cliout

import (
	"encoding/json"
	"fmt"
	"io"
)

type Format string

const (
	FormatText Format = "text"
	FormatJSON Format = "json"
)

func ParseFormat(raw string) (Format, error) {
	switch Format(raw) {
	case "", FormatText:
		return FormatText, nil
	case FormatJSON:
		return FormatJSON, nil
	default:
		return "", fmt.Errorf("unsupported output format %q: must be one of: text, json", raw)
	}
}

// WriteJSON writes v to w as pretty-printed JSON with 2-space indentation.
func WriteJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
