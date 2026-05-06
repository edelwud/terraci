package pipeline

const artifactNamePrefix = "terraci-"

// ArtifactNamePattern matches all Terraci artifacts in providers that support
// pattern-based downloads.
func ArtifactNamePattern() string {
	return artifactNamePrefix + "*"
}

// PlanArtifactName returns the artifact name used by a plan job.
func PlanArtifactName(jobName string) string {
	return artifactNamePrefix + jobName
}

// PlanArtifact returns the artifact published by a plan job.
func PlanArtifact(jobName string, paths []string) Artifact {
	return Artifact{
		Name:  PlanArtifactName(jobName),
		Paths: compactArtifactPaths(paths),
	}
}

// ResultArtifactName returns the artifact name used by a contributed result job.
func ResultArtifactName(jobName string) string {
	return artifactNamePrefix + jobName + "-results"
}

// ResultArtifact returns the artifact published by a contributed result job.
func ResultArtifact(jobName string, paths ...string) Artifact {
	return Artifact{
		Name:  ResultArtifactName(jobName),
		Paths: compactArtifactPaths(paths),
	}
}

func compactArtifactPaths(paths []string) []string {
	if len(paths) == 0 {
		return nil
	}

	result := make([]string, 0, len(paths))
	for _, path := range paths {
		if path == "" {
			continue
		}
		result = append(result, path)
	}
	return result
}
