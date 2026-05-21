package cmd

import (
	"github.com/edelwud/terraci/cmd/terraci/internal/runflow"
	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/plugin/registry"
)

// App holds root command flags and process-wide command services.
type App struct {
	WorkDir string

	// Global flag values
	cfgFile  string
	logLevel string

	// Version info
	Version string
	Commit  string
	Date    string

	// reports is the command report store. AppContexts created for each command
	// share this store so file-backed artifacts and mid-command report exchange
	// use the same boundary.
	reports ci.ReportStore
}

func newApp(version, commit, date string) *App {
	return &App{
		Version: version,
		Commit:  commit,
		Date:    date,
	}
}

func (a *App) newRunFlow() *runflow.Flow {
	return runflow.New(runflow.Options{
		RegistryFactory: registry.New,
		InitLogger:      initLogger,
		SetLogLevel:     setLogLevelFromString,
		Version:         a.Version,
		Reports:         a.reports,
	})
}
