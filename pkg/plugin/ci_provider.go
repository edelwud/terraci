package plugin

import (
	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/pipeline"
)

// --- CI Provider (split into focused interfaces) ---

// EnvDetector detects whether this plugin's CI environment is active.
type EnvDetector interface {
	Plugin
	DetectEnv() bool
}

// CIInfoProvider provides CI-specific metadata.
type CIInfoProvider interface {
	Plugin
	ProviderName() string
	PipelineID() string
	CommitSHA() string
}

// PipelineGeneratorFactory creates pipeline generators bound to a pre-built
// IR. Core builds the IR once via pipeline.Build(opts) and passes it here, so
// providers do not need depGraph, target modules, or contributions — they
// only render the IR.
type PipelineGeneratorFactory interface {
	Plugin
	NewGenerator(ctx *AppContext, ir *pipeline.IR) pipeline.Generator
}

// CommentServiceFactory creates PR/MR comment services.
type CommentServiceFactory interface {
	Plugin
	NewCommentService(ctx *AppContext) ci.CommentService
}

// ResolvedCIProvider is the resolved CI provider, assembled from the focused interfaces above.
// Returned by ResolveCIProvider(). CommentServiceFactory is optional — not all CI providers
// support PR/MR comments.
type ResolvedCIProvider struct {
	plugin   Plugin
	metadata CIInfoProvider
	gen      PipelineGeneratorFactory
	comment  CommentServiceFactory // may be nil
}

// NewResolvedCIProvider constructs a ResolvedCIProvider. The comment parameter may be nil for
// CI providers that do not support PR/MR comments.
func NewResolvedCIProvider(p Plugin, meta CIInfoProvider, gen PipelineGeneratorFactory, comment CommentServiceFactory) *ResolvedCIProvider {
	return &ResolvedCIProvider{plugin: p, metadata: meta, gen: gen, comment: comment}
}

func (c *ResolvedCIProvider) Name() string         { return c.plugin.Name() }
func (c *ResolvedCIProvider) Description() string  { return c.plugin.Description() }
func (c *ResolvedCIProvider) ProviderName() string { return c.metadata.ProviderName() }
func (c *ResolvedCIProvider) PipelineID() string   { return c.metadata.PipelineID() }
func (c *ResolvedCIProvider) CommitSHA() string    { return c.metadata.CommitSHA() }

// NewGenerator returns a pipeline generator bound to the supplied IR.
func (c *ResolvedCIProvider) NewGenerator(ctx *AppContext, ir *pipeline.IR) pipeline.Generator {
	return c.gen.NewGenerator(ctx, ir)
}

// NewCommentService returns the comment service and true, or nil and false
// if the CI provider does not support PR/MR comments.
func (c *ResolvedCIProvider) NewCommentService(ctx *AppContext) (ci.CommentService, bool) {
	if c.comment == nil {
		return nil, false
	}
	return c.comment.NewCommentService(ctx), true
}

// Plugin returns the underlying plugin instance.
func (c *ResolvedCIProvider) Plugin() Plugin { return c.plugin }
