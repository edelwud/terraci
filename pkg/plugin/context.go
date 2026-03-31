package plugin

import (
	"github.com/edelwud/terraci/pkg/config"
	"github.com/edelwud/terraci/pkg/log"
)

// AppContext is the public API available to plugins.
//
// ServiceDir is the resolved absolute path — use it for runtime file I/O.
// For pipeline artifact paths (CI templates), use Config.ServiceDir which
// preserves the original relative value from .terraci.yaml.
//
// After initialization, call Freeze() to prevent the framework from refreshing
// this shared context from App state again during the current command run.
type AppContext struct {
	config     *config.Config
	workDir    string
	serviceDir string // resolved absolute path to project service directory
	version    string
	reports    *ReportRegistry

	frozen bool
}

// NewAppContext creates a framework-managed plugin context.
func NewAppContext(cfg *config.Config, workDir, serviceDir, version string, reports *ReportRegistry) *AppContext {
	if reports == nil {
		reports = NewReportRegistry()
	}
	ctx := &AppContext{reports: reports}
	ctx.Update(cfg, workDir, serviceDir, version)
	return ctx
}

// Update refreshes the framework-managed view of app state until the context is frozen.
func (ctx *AppContext) Update(cfg *config.Config, workDir, serviceDir, version string) {
	if ctx.frozen {
		log.Debug("AppContext.Update called after Freeze — ignored")
		return
	}
	ctx.config = cfg.Clone()
	ctx.workDir = workDir
	ctx.serviceDir = serviceDir
	ctx.version = version
	if ctx.reports == nil {
		ctx.reports = NewReportRegistry()
	}
}

// Config returns a defensive copy of the loaded TerraCi configuration.
// For repeated access within a single use-case, cache the result locally:
//
//	cfg := appCtx.Config()
//	// use cfg throughout the function
//
// This avoids repeated deep copies while preserving immutability guarantees.
func (ctx *AppContext) Config() *config.Config {
	return ctx.config.Clone()
}

// WorkDir returns the working directory for the current command.
func (ctx *AppContext) WorkDir() string {
	return ctx.workDir
}

// ServiceDir returns the resolved absolute service directory path.
func (ctx *AppContext) ServiceDir() string {
	return ctx.serviceDir
}

// Version returns the current TerraCi version string.
func (ctx *AppContext) Version() string {
	return ctx.version
}

// Reports returns the shared in-process report registry.
func (ctx *AppContext) Reports() *ReportRegistry {
	return ctx.reports
}

// Freeze marks the context as final for framework-managed updates.
func (ctx *AppContext) Freeze() {
	ctx.frozen = true
}

// IsFrozen returns whether the context has been frozen.
func (ctx *AppContext) IsFrozen() bool {
	return ctx.frozen
}
