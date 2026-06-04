package runflow

import (
	"context"
	"errors"

	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/config"
	"github.com/edelwud/terraci/pkg/diagnostic"
	"github.com/edelwud/terraci/pkg/pipeline"
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
	contribs    []*pipeline.Contribution
}

func newPrepared(ctx context.Context, appCtx *plugin.AppContext, plugins *registry.Registry, cfg *config.Config, workDir string, reports ci.ReportStore, contribs []*pipeline.Contribution) (*Prepared, error) {
	binding, err := plugin.NewCommandBinding(plugin.CommandBindingOptions{
		AppContext:            appCtx,
		Source:                plugins,
		PipelineContributions: contribs,
	})
	if err != nil {
		return nil, err
	}
	prepared := &Prepared{
		appCtx:   appCtx,
		registry: plugins,
		config:   config.NewSnapshot(cfg),
		loaded:   cfg.Clone(),
		workDir:  workDir,
		reports:  reports,
		contribs: cloneContributions(contribs),
	}
	prepared.ctx = context.WithValue(plugin.BindCommandContext(ctx, binding), preparedContextKey{}, prepared)
	return prepared, nil
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

// ExtensionSchemas returns extension schema samples from the command-scoped
// plugin snapshot.
func (p *Prepared) ExtensionSchemas() map[string]any {
	if p == nil {
		return nil
	}
	return p.registry.ExtensionSchemas()
}

// VersionSnapshot returns version metadata from the command-scoped plugin
// snapshot.
func (p *Prepared) VersionSnapshot() registry.VersionSnapshot {
	if p == nil {
		return registry.VersionSnapshot{}
	}
	return p.registry.VersionSnapshot()
}

// InitWizardSnapshot returns init wizard bindings from the command-scoped
// plugin snapshot.
func (p *Prepared) InitWizardSnapshot() (*registry.InitWizardSnapshot, error) {
	if p == nil {
		return &registry.InitWizardSnapshot{}, nil
	}
	return p.registry.InitWizardSnapshot()
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

// PipelineContributions returns the command-scoped pipeline contribution
// snapshot collected by runflow.
func (p *Prepared) PipelineContributions() []*pipeline.Contribution {
	if p == nil {
		return nil
	}
	return cloneContributions(p.contribs)
}

// Diagnostics returns non-fatal diagnostics produced while preparing the command.
func (p *Prepared) Diagnostics() diagnostic.List {
	if p == nil {
		return diagnostic.List{}
	}
	return p.diagnostics
}

func cloneContributions(contribs []*pipeline.Contribution) []*pipeline.Contribution {
	if len(contribs) == 0 {
		return nil
	}
	clone := make([]*pipeline.Contribution, len(contribs))
	for i, contribution := range contribs {
		clone[i] = contribution.Clone()
	}
	return clone
}
