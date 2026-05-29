package runflow

import (
	"context"
	"errors"

	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/config"
	"github.com/edelwud/terraci/pkg/diagnostic"
	"github.com/edelwud/terraci/pkg/plugin"
	"github.com/edelwud/terraci/pkg/plugin/registry"
)

var errPreparedNotBound = errors.New("runflow prepared state is not bound")

type preparedContextKey struct{}

// Prepared is the immutable command-scoped state produced by Flow.Prepare.
type Prepared struct {
	ctx         context.Context
	appCtx      *plugin.AppContext
	registry    *registry.Registry
	config      config.Snapshot
	loaded      *config.Config
	workDir     string
	reports     ci.ReportStore
	diagnostics diagnostic.List
}

func newPrepared(ctx context.Context, appCtx *plugin.AppContext, plugins *registry.Registry, cfg *config.Config, workDir string, reports ci.ReportStore) *Prepared {
	prepared := &Prepared{
		appCtx:   appCtx,
		registry: plugins,
		config:   config.NewSnapshot(cfg),
		loaded:   cfg.Clone(),
		workDir:  workDir,
		reports:  reports,
	}
	prepared.ctx = context.WithValue(plugin.WithContext(ctx, appCtx), preparedContextKey{}, prepared)
	return prepared
}

// FromContext returns the Prepared command state bound by Flow.Prepare.
func FromContext(ctx context.Context) (*Prepared, error) {
	if ctx == nil {
		return nil, errPreparedNotBound
	}
	prepared, ok := ctx.Value(preparedContextKey{}).(*Prepared)
	if !ok || prepared == nil {
		return nil, errPreparedNotBound
	}
	return prepared, nil
}

// Context returns a context containing both Prepared and plugin.AppContext.
func (p *Prepared) Context() context.Context {
	if p == nil || p.ctx == nil {
		return context.Background()
	}
	return p.ctx
}

// AppContext returns the plugin SDK context for this command run.
func (p *Prepared) AppContext() *plugin.AppContext {
	if p == nil {
		return nil
	}
	return p.appCtx
}

// Registry returns the command-scoped plugin registry snapshot.
func (p *Prepared) Registry() *registry.Registry {
	if p == nil {
		return nil
	}
	return p.registry
}

// Config returns the loaded immutable TerraCi config snapshot.
func (p *Prepared) Config() config.Snapshot {
	if p == nil {
		return config.Snapshot{}
	}
	return p.config
}

// LoadedConfig returns a defensive mutable copy of the loaded config.
func (p *Prepared) LoadedConfig() *config.Config {
	if p == nil {
		return nil
	}
	return p.loaded.Clone()
}

// WorkDir returns the command working directory.
func (p *Prepared) WorkDir() string {
	if p == nil {
		return ""
	}
	return p.workDir
}

// Reports returns the command report store.
func (p *Prepared) Reports() ci.ReportStore {
	if p == nil {
		return nil
	}
	return p.reports
}

// Diagnostics returns non-fatal diagnostics produced while preparing the command.
func (p *Prepared) Diagnostics() diagnostic.List {
	if p == nil {
		return diagnostic.List{}
	}
	return p.diagnostics
}
