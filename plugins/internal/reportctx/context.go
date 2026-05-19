package reportctx

import (
	"time"

	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/plugin"
)

// Options carries optional report artifact metadata not always available from
// AppContext alone.
type Options struct {
	Collection  *ci.PlanResultCollection
	Fingerprint string
	GeneratedAt time.Time
}

// FromApp builds a canonical CI artifact context from the plugin AppContext.
// CI provider metadata is best-effort: report production must not fail merely
// because a command runs outside CI or without a configured provider.
func FromApp(appCtx *plugin.AppContext, opts Options) ci.ArtifactContext {
	if appCtx == nil {
		return ci.NewArtifactContext(ci.ArtifactContextOptions{
			PlanResultsFingerprint: fingerprint(opts),
			GeneratedAt:            opts.GeneratedAt,
		})
	}

	commitSHA, pipelineID := collectionMetadata(opts.Collection)
	if resolver := appCtx.Resolver(); resolver != nil {
		provider, err := resolver.ResolveCIProvider()
		if err == nil && provider != nil {
			if value := provider.CommitSHA(); value != "" {
				commitSHA = value
			}
			if value := provider.PipelineID(); value != "" {
				pipelineID = value
			}
		}
	}

	return ci.NewArtifactContext(ci.ArtifactContextOptions{
		ServiceDir:             appCtx.ServiceDir(),
		WorkDir:                appCtx.WorkDir(),
		CommitSHA:              commitSHA,
		PipelineID:             pipelineID,
		PlanResultsFingerprint: fingerprint(opts),
		GeneratedAt:            opts.GeneratedAt,
	})
}

func fingerprint(opts Options) string {
	if opts.Fingerprint != "" {
		return opts.Fingerprint
	}
	if opts.Collection != nil {
		return opts.Collection.Fingerprint()
	}
	return ""
}

func collectionMetadata(collection *ci.PlanResultCollection) (commitSHA, pipelineID string) {
	if collection == nil {
		return "", ""
	}
	return collection.CommitSHA, collection.PipelineID
}
