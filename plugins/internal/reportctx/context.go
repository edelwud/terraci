package reportctx

import (
	"time"

	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/plugin"
)

// Options carries report artifact metadata not always available from AppContext
// alone.
type Options struct {
	Producer    string
	Collection  *ci.PlanResultCollection
	Fingerprint string
	GeneratedAt time.Time
}

// NewRun builds a canonical CI artifact run from the plugin AppContext. CI
// provider metadata is best-effort: report production must not fail merely
// because a command runs outside CI or without a configured provider.
func NewRun(appCtx *plugin.AppContext, opts Options) (ci.ArtifactRun, error) {
	if appCtx == nil {
		artifact := ci.NewArtifactContext(ci.ArtifactContextOptions{
			PlanResultsFingerprint: fingerprint(opts),
			GeneratedAt:            opts.GeneratedAt,
		})
		return ci.NewArtifactRun(ci.ArtifactRunOptions{
			Producer:    opts.Producer,
			Artifact:    artifact,
			PlanResults: opts.Collection,
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

	artifact := ci.NewArtifactContext(ci.ArtifactContextOptions{
		ServiceDir:             appCtx.ServiceDir(),
		WorkDir:                appCtx.WorkDir(),
		CommitSHA:              commitSHA,
		PipelineID:             pipelineID,
		PlanResultsFingerprint: fingerprint(opts),
		GeneratedAt:            opts.GeneratedAt,
	})
	return ci.NewArtifactRun(ci.ArtifactRunOptions{
		Producer:    opts.Producer,
		Artifact:    artifact,
		PlanResults: opts.Collection,
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
