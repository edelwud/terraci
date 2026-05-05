package cmd

import (
	"testing"

	"github.com/edelwud/terraci/pkg/plugin/registry"
)

func TestAppBuildContextReturnsFreshContextEachTime(t *testing.T) {
	app := newApp("test", "abc", "today")

	first := app.BuildContext()
	if first.Resolver() == nil {
		t.Fatal("BuildContext should bind a resolver")
	}

	app.ResetPluginsForCommand()
	second := app.BuildContext()

	if second == first {
		t.Fatal("BuildContext should return a fresh AppContext per call")
	}
	if second.Reports() != first.Reports() {
		t.Fatal("BuildContext should keep the long-lived reports registry across runs")
	}
	if _, ok := second.Resolver().(*registry.Registry); !ok {
		t.Fatalf("AppContext resolver = %T, want *registry.Registry", second.Resolver())
	}
}
