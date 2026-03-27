package ci

import "testing"

func TestDetectPipelineID(t *testing.T) {
	t.Setenv("CI_PIPELINE_ID", "")
	t.Setenv("GITHUB_RUN_ID", "12345")

	got := DetectPipelineID()
	if got != "12345" {
		t.Errorf("DetectPipelineID() = %q, want %q", got, "12345")
	}
}

func TestDetectPipelineID_GitLab(t *testing.T) {
	t.Setenv("CI_PIPELINE_ID", "67890")
	t.Setenv("GITHUB_RUN_ID", "12345")

	got := DetectPipelineID()
	if got != "67890" {
		t.Errorf("DetectPipelineID() = %q, want %q (GitLab takes precedence)", got, "67890")
	}
}

func TestDetectCommitSHA(t *testing.T) {
	t.Setenv("CI_COMMIT_SHA", "")
	t.Setenv("GITHUB_SHA", "abc123")

	got := DetectCommitSHA()
	if got != "abc123" {
		t.Errorf("DetectCommitSHA() = %q, want %q", got, "abc123")
	}
}

func TestDetectCommitSHA_GitLab(t *testing.T) {
	t.Setenv("CI_COMMIT_SHA", "def456")
	t.Setenv("GITHUB_SHA", "abc123")

	got := DetectCommitSHA()
	if got != "def456" {
		t.Errorf("DetectCommitSHA() = %q, want %q (GitLab takes precedence)", got, "def456")
	}
}
