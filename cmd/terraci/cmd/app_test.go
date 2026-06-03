package cmd

import (
	"context"
	"testing"

	"github.com/edelwud/terraci/cmd/terraci/internal/runflow"
)

func TestAppRunFlowReturnsFreshContextEachTime(t *testing.T) {
	app := newApp("test", "abc", "today")

	first, err := app.newRunFlow().Prepare(context.Background(), runFlowTestRequest(app))
	if err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}

	app.reports = first.Reports()

	second, err := app.newRunFlow().Prepare(context.Background(), runFlowTestRequest(app))
	if err != nil {
		t.Fatalf("Prepare() second error = %v", err)
	}

	if second.AppContext() == first.AppContext() {
		t.Fatal("Prepare should return a fresh AppContext per call")
	}
	if second.AppContext().Reports() != first.AppContext().Reports() {
		t.Fatal("Prepare should keep the long-lived reports registry across runs")
	}
}

func runFlowTestRequest(app *App) runflow.Request {
	return runflow.Request{
		CommandName: "version",
		WorkDir:     app.WorkDir,
		LogLevel:    "info",
		Policy:      runflow.CommandPolicy{SkipConfig: true},
	}
}
