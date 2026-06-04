package plugin

import (
	"time"

	"github.com/edelwud/terraci/pkg/ci"
)

// ArtifactRunOptions carries report artifact metadata not always available
// from AppContext alone.
type ArtifactRunOptions struct {
	Producer    string
	Collection  *ci.PlanResultCollection
	Fingerprint string
	GeneratedAt time.Time
}

// NewArtifactRun builds a canonical CI artifact run from AppContext. CI
// provider metadata is best-effort: report production must not fail merely
// because a command runs outside CI or without a configured provider.
func NewArtifactRun(appCtx *AppContext, opts ArtifactRunOptions) (ci.ArtifactRun, error) {
	artifactOpts := ci.ArtifactContextOptions{
		PlanResultsFingerprint: artifactRunFingerprint(opts),
		GeneratedAt:            opts.GeneratedAt,
	}

	if appCtx != nil {
		commitSHA, pipelineID := artifactRunCollectionMetadata(opts.Collection)
		provider, err := appCtx.CIResolver().ResolveCIProvider()
		if err == nil && provider != nil {
			if value := provider.CommitSHA(); value != "" {
				commitSHA = value
			}
			if value := provider.PipelineID(); value != "" {
				pipelineID = value
			}
		}

		artifactOpts.ServiceDir = appCtx.ServiceDir()
		artifactOpts.WorkDir = appCtx.WorkDir()
		artifactOpts.CommitSHA = commitSHA
		artifactOpts.PipelineID = pipelineID
	}

	return ci.NewArtifactRun(ci.ArtifactRunOptions{
		Producer:    opts.Producer,
		Artifact:    ci.NewArtifactContext(artifactOpts),
		PlanResults: opts.Collection,
	})
}

func artifactRunFingerprint(opts ArtifactRunOptions) string {
	if opts.Fingerprint != "" {
		return opts.Fingerprint
	}
	if opts.Collection != nil {
		return opts.Collection.Fingerprint()
	}
	return ""
}

func artifactRunCollectionMetadata(collection *ci.PlanResultCollection) (commitSHA, pipelineID string) {
	if collection == nil {
		return "", ""
	}
	return collection.CommitSHA, collection.PipelineID
}
