package plugin

import (
	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/discovery"
	"github.com/edelwud/terraci/pkg/graph"
	"github.com/edelwud/terraci/pkg/pipeline"
)

// --- CI Provider (split into focused interfaces) ---

// EnvDetector detects whether this plugin's CI environment is active.
type EnvDetector interface {
	Plugin
	DetectEnv() bool
}

// CIMetadata provides CI-specific metadata.
type CIMetadata interface {
	Plugin
	ProviderName() string
	PipelineID() string
	CommitSHA() string
}

// GeneratorFactory creates pipeline generators.
type GeneratorFactory interface {
	Plugin
	NewGenerator(ctx *AppContext, depGraph *graph.DependencyGraph, modules []*discovery.Module) pipeline.Generator
}

// CommentFactory creates PR/MR comment services.
type CommentFactory interface {
	Plugin
	NewCommentService(ctx *AppContext) ci.CommentService
}

// CIProvider is the resolved CI provider, assembled from the focused interfaces above.
// Returned by ResolveProvider(). CommentFactory is optional — not all CI providers
// support PR/MR comments.
type CIProvider struct {
	plugin   Plugin
	metadata CIMetadata
	gen      GeneratorFactory
	comment  CommentFactory // may be nil
}

// NewCIProvider constructs a CIProvider. The comment parameter may be nil for
// CI providers that do not support PR/MR comments.
func NewCIProvider(p Plugin, meta CIMetadata, gen GeneratorFactory, comment CommentFactory) *CIProvider {
	return &CIProvider{plugin: p, metadata: meta, gen: gen, comment: comment}
}

func (c *CIProvider) Name() string         { return c.plugin.Name() }
func (c *CIProvider) Description() string  { return c.plugin.Description() }
func (c *CIProvider) ProviderName() string { return c.metadata.ProviderName() }
func (c *CIProvider) PipelineID() string   { return c.metadata.PipelineID() }
func (c *CIProvider) CommitSHA() string    { return c.metadata.CommitSHA() }

func (c *CIProvider) NewGenerator(ctx *AppContext, depGraph *graph.DependencyGraph, modules []*discovery.Module) pipeline.Generator {
	return c.gen.NewGenerator(ctx, depGraph, modules)
}

// NewCommentService returns the comment service and true, or nil and false
// if the CI provider does not support PR/MR comments.
func (c *CIProvider) NewCommentService(ctx *AppContext) (ci.CommentService, bool) {
	if c.comment == nil {
		return nil, false
	}
	return c.comment.NewCommentService(ctx), true
}

// Plugin returns the underlying plugin instance.
func (c *CIProvider) Plugin() Plugin { return c.plugin }
