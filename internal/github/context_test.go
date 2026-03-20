package github

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectPRContext_InPR(t *testing.T) {
	t.Setenv("GITHUB_REPOSITORY", "owner/repo")
	t.Setenv("GITHUB_REF", "refs/pull/42/merge")
	t.Setenv("GITHUB_SHA", "abc123")
	t.Setenv("GITHUB_RUN_ID", "12345")
	t.Setenv("GITHUB_HEAD_REF", "")
	t.Setenv("GITHUB_BASE_REF", "")
	t.Setenv("GITHUB_EVENT_PATH", "")

	ctx := DetectPRContext()

	if !ctx.InPR {
		t.Error("expected InPR to be true")
	}
	if ctx.PRNumber != 42 {
		t.Errorf("expected PRNumber 42, got %d", ctx.PRNumber)
	}
	if ctx.Owner != "owner" {
		t.Errorf("expected Owner %q, got %q", "owner", ctx.Owner)
	}
	if ctx.Repo != "repo" {
		t.Errorf("expected Repo %q, got %q", "repo", ctx.Repo)
	}
	if ctx.CommitSHA != "abc123" {
		t.Errorf("expected CommitSHA %q, got %q", "abc123", ctx.CommitSHA)
	}
	if ctx.RunID != "12345" {
		t.Errorf("expected RunID %q, got %q", "12345", ctx.RunID)
	}
}

func TestDetectPRContext_NotInPR(t *testing.T) {
	t.Setenv("GITHUB_REPOSITORY", "owner/repo")
	t.Setenv("GITHUB_REF", "refs/heads/main")
	t.Setenv("GITHUB_SHA", "")
	t.Setenv("GITHUB_RUN_ID", "")
	t.Setenv("GITHUB_HEAD_REF", "")
	t.Setenv("GITHUB_BASE_REF", "")
	t.Setenv("GITHUB_EVENT_PATH", "")

	ctx := DetectPRContext()

	if ctx.InPR {
		t.Error("expected InPR to be false for non-PR ref")
	}
	if ctx.PRNumber != 0 {
		t.Errorf("expected PRNumber 0, got %d", ctx.PRNumber)
	}
}

func TestDetectPRContext_BranchInfo(t *testing.T) {
	t.Setenv("GITHUB_REPOSITORY", "owner/repo")
	t.Setenv("GITHUB_REF", "refs/pull/10/merge")
	t.Setenv("GITHUB_SHA", "")
	t.Setenv("GITHUB_RUN_ID", "")
	t.Setenv("GITHUB_HEAD_REF", "feature-branch")
	t.Setenv("GITHUB_BASE_REF", "main")
	t.Setenv("GITHUB_EVENT_PATH", "")

	ctx := DetectPRContext()

	if ctx.SourceBranch != "feature-branch" {
		t.Errorf("expected SourceBranch %q, got %q", "feature-branch", ctx.SourceBranch)
	}
	if ctx.TargetBranch != "main" {
		t.Errorf("expected TargetBranch %q, got %q", "main", ctx.TargetBranch)
	}
}

func TestGetPRNumberFromEvent(t *testing.T) {
	tmpDir := t.TempDir()
	eventFile := filepath.Join(tmpDir, "event.json")

	err := os.WriteFile(eventFile, []byte(`{"pull_request":{"number":99}}`), 0o644)
	if err != nil {
		t.Fatalf("failed to write temp event file: %v", err)
	}

	t.Setenv("GITHUB_EVENT_PATH", eventFile)

	num := getPRNumberFromEvent()
	if num != 99 {
		t.Errorf("expected PR number 99, got %d", num)
	}
}

func TestGetPRNumberFromEvent_NoFile(t *testing.T) {
	t.Setenv("GITHUB_EVENT_PATH", "")

	num := getPRNumberFromEvent()
	if num != 0 {
		t.Errorf("expected PR number 0 when no event file, got %d", num)
	}
}

func TestGetPRNumberFromEvent_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	eventFile := filepath.Join(tmpDir, "event.json")

	err := os.WriteFile(eventFile, []byte(`not json`), 0o644)
	if err != nil {
		t.Fatalf("failed to write temp event file: %v", err)
	}

	t.Setenv("GITHUB_EVENT_PATH", eventFile)

	num := getPRNumberFromEvent()
	if num != 0 {
		t.Errorf("expected PR number 0 for invalid JSON, got %d", num)
	}
}

func TestGetPRNumberFromEvent_NoPullRequest(t *testing.T) {
	tmpDir := t.TempDir()
	eventFile := filepath.Join(tmpDir, "event.json")

	err := os.WriteFile(eventFile, []byte(`{"action":"push"}`), 0o644)
	if err != nil {
		t.Fatalf("failed to write temp event file: %v", err)
	}

	t.Setenv("GITHUB_EVENT_PATH", eventFile)

	num := getPRNumberFromEvent()
	if num != 0 {
		t.Errorf("expected PR number 0 when no pull_request in event, got %d", num)
	}
}

func TestGetPRNumberFromEvent_NonexistentFile(t *testing.T) {
	t.Setenv("GITHUB_EVENT_PATH", "/nonexistent/path/event.json")

	num := getPRNumberFromEvent()
	if num != 0 {
		t.Errorf("expected PR number 0 for nonexistent file, got %d", num)
	}
}

func TestDetectPRContext_FromEventPayload(t *testing.T) {
	tmpDir := t.TempDir()
	eventFile := filepath.Join(tmpDir, "event.json")

	err := os.WriteFile(eventFile, []byte(`{"pull_request":{"number":77}}`), 0o644)
	if err != nil {
		t.Fatalf("failed to write temp event file: %v", err)
	}

	t.Setenv("GITHUB_REPOSITORY", "owner/repo")
	t.Setenv("GITHUB_REF", "refs/heads/feature") // not a PR ref
	t.Setenv("GITHUB_SHA", "def456")
	t.Setenv("GITHUB_RUN_ID", "99999")
	t.Setenv("GITHUB_HEAD_REF", "feature")
	t.Setenv("GITHUB_BASE_REF", "main")
	t.Setenv("GITHUB_EVENT_PATH", eventFile)

	ctx := DetectPRContext()

	if !ctx.InPR {
		t.Error("expected InPR to be true from event payload")
	}
	if ctx.PRNumber != 77 {
		t.Errorf("expected PRNumber 77, got %d", ctx.PRNumber)
	}
}

func TestDetectPRContext_InvalidRefParts(t *testing.T) {
	t.Setenv("GITHUB_REPOSITORY", "owner/repo")
	t.Setenv("GITHUB_REF", "refs/pull/notanumber/merge")
	t.Setenv("GITHUB_SHA", "")
	t.Setenv("GITHUB_RUN_ID", "")
	t.Setenv("GITHUB_HEAD_REF", "")
	t.Setenv("GITHUB_BASE_REF", "")
	t.Setenv("GITHUB_EVENT_PATH", "")

	ctx := DetectPRContext()

	if ctx.InPR {
		t.Error("expected InPR to be false when PR number is not a valid integer")
	}
}
