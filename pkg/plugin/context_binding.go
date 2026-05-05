package plugin

import (
	"github.com/edelwud/terraci/pkg/config"
	"github.com/edelwud/terraci/pkg/log"
)

// Update refreshes the framework-managed view of app state until the context is frozen.
func (ctx *AppContext) Update(cfg *config.Config, workDir, serviceDir, version string) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()

	if ctx.frozen {
		log.Debug("AppContext.Update called after Freeze — ignored")
		return
	}
	ctx.config = cfg
	ctx.workDir = workDir
	ctx.serviceDir = serviceDir
	ctx.version = version
	if ctx.reports == nil {
		ctx.reports = NewReportRegistry()
	}
}

// SetResolver binds the per-run plugin resolver. Framework code calls this
// before plugins receive the context.
func (ctx *AppContext) SetResolver(resolver Resolver) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()

	if ctx.frozen {
		log.Debug("AppContext.SetResolver called after Freeze — ignored")
		return
	}
	ctx.resolver = resolver
}

// BeginCommand reopens the framework-managed context for a new command run and
// binds it to that run's fresh plugin resolver.
func (ctx *AppContext) BeginCommand(resolver Resolver) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()

	ctx.frozen = false
	ctx.resolver = resolver
}

// Freeze marks the context as final for framework-managed updates.
func (ctx *AppContext) Freeze() {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()

	ctx.frozen = true
}

// IsFrozen returns whether the context has been frozen.
func (ctx *AppContext) IsFrozen() bool {
	ctx.mu.RLock()
	defer ctx.mu.RUnlock()

	return ctx.frozen
}
