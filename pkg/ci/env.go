package ci

import "os"

// DetectPipelineID reads pipeline ID from CI environment.
func DetectPipelineID() string {
	if v := os.Getenv("CI_PIPELINE_ID"); v != "" {
		return v
	}
	return os.Getenv("GITHUB_RUN_ID")
}

// DetectCommitSHA reads commit SHA from CI environment.
func DetectCommitSHA() string {
	if v := os.Getenv("CI_COMMIT_SHA"); v != "" {
		return v
	}
	return os.Getenv("GITHUB_SHA")
}
