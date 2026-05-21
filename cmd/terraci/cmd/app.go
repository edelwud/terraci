package cmd

import (
	"github.com/edelwud/terraci/cmd/terraci/internal/runflow"
	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/config"
	"github.com/edelwud/terraci/pkg/plugin/registry"
)

// App holds shared state for all commands, replacing package-level globals.
//
// Plugins is the per-command plugin registry, rebuilt fresh in PreRunE for
// every cobra command run. Code that runs *before* PreRunE (root command
// registration in NewRootCmd) holds an initial registry created in newApp;
// after PreRunE that initial registry is replaced by a freshly built one
// and the AppContext is constructed once and attached to cmd.Context().
type App struct {
	Config  *config.Config
	WorkDir string
	Plugins *registry.Registry

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
		Plugins: registry.New(),
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
