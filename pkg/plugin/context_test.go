package plugin

import (
	"testing"

	"github.com/edelwud/terraci/pkg/ci"
)

func TestNewExecutionContext(t *testing.T) {
	collection := &ci.PlanResultCollection{}
	ctx := NewExecutionContext(collection)
	if ctx.PlanResults != collection {
		t.Fatal("PlanResults should match")
	}
	if ctx.Data == nil {
		t.Fatal("Data map should be initialized")
	}
}

func TestExecutionContext_SetGetData(t *testing.T) {
	ctx := NewExecutionContext(nil)
	ctx.SetData("key", "value")
	v, ok := ctx.GetData("key")
	if !ok || v != "value" {
		t.Fatalf("expected value, got %v (ok=%v)", v, ok)
	}
}
