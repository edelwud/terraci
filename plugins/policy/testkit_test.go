package policy

import (
	"bytes"
	"testing"

	rawlog "github.com/caarlos0/log"
)

func capturePolicyTextOutput(t *testing.T, fn func()) string {
	t.Helper()
	oldLogger := rawlog.Log
	var buf bytes.Buffer
	rawlog.Log = rawlog.New(&buf)
	defer func() { rawlog.Log = oldLogger }()
	fn()
	return buf.String()
}
