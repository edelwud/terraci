package cmd

import (
	"context"
	"testing"

	"github.com/edelwud/terraci/cmd/terraci/internal/runflow"
	"github.com/edelwud/terraci/pkg/plugin/registry"
)

func TestAppRunFlowReturnsFreshContextEachTime(t *testing.T) {
	app := newApp("test", "abc", "today")

	first, err := app.newRunFlow().Prepare(context.Background(), runFlowTestRequest(app))
	if err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}

	app.Plugins = first.Registry
	app.reports = first.Reports

	second, err := app.newRunFlow().Prepare(context.Background(), runFlowTestRequest(app))
	if err != nil {
		t.Fatalf("Prepare() second error = %v", err)
	}

	if second.AppContext == first.AppContext {
		t.Fatal("Prepare should return a fresh AppContext per call")
	}
	if second.AppContext.Reports() != first.AppContext.Reports() {
		t.Fatal("Prepare should keep the long-lived reports registry across runs")
	}
	if _, ok := second.AppContext.Resolver().(*registry.Registry); !ok {
		t.Fatalf("AppContext resolver = %T, want *registry.Registry", second.AppContext.Resolver())
	}
}

func runFlowTestRequest(app *App) runflow.Request {
	return runflow.Request{
		CommandName: "version",
		WorkDir:     app.WorkDir,
		LogLevel:    "info",
		SkipConfig:  true,
	}
}
