package storage

import (
	"testing"

	"github.com/edelwud/terraci/plugins/cost/internal/resourcedef"
)

func parsedAttrs(tb testing.TB, def resourcedef.Definition, attrs map[string]any) resourcedef.Attributes {
	tb.Helper()
	parsed, err := def.ParseAttrs(resourcedef.NewRawAttrs(attrs))
	if err != nil {
		tb.Fatalf("ParseAttrs() error = %v", err)
	}
	return parsed
}

func rawAttrs(attrs map[string]any) resourcedef.RawAttrs {
	return resourcedef.NewRawAttrs(attrs)
}
