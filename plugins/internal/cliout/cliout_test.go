package cliout

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestParseFormat(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		raw     string
		want    Format
		wantErr string
	}{
		{name: "empty defaults to text", raw: "", want: FormatText},
		{name: "text", raw: "text", want: FormatText},
		{name: "json", raw: "json", want: FormatJSON},
		{name: "invalid", raw: "yaml", wantErr: `unsupported output format "yaml"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseFormat(tt.raw)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatal("ParseFormat() error = nil, want error")
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("ParseFormat() error = %q, want %q", err.Error(), tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseFormat() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("ParseFormat() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestWriteJSON(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	if err := WriteJSON(&buf, map[string]string{"ok": "true"}); err != nil {
		t.Fatalf("WriteJSON() error = %v", err)
	}
	if !json.Valid(buf.Bytes()) {
		t.Fatalf("WriteJSON() output is not JSON: %s", buf.String())
	}
	if !strings.Contains(buf.String(), "\n  ") {
		t.Fatalf("WriteJSON() output is not indented: %s", buf.String())
	}
}
