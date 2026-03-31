package mr

import "testing"

func TestDetectContext(t *testing.T) {
	t.Run("not in MR", func(t *testing.T) {
		t.Setenv("CI_MERGE_REQUEST_IID", "")

		ctx := DetectContext()
		if ctx.InMR {
			t.Fatal("expected InMR to be false")
		}
	})

	t.Run("in MR", func(t *testing.T) {
		t.Setenv("CI_PROJECT_ID", "12345")
		t.Setenv("CI_PROJECT_PATH", "group/project")
		t.Setenv("CI_MERGE_REQUEST_IID", "42")
		t.Setenv("CI_MERGE_REQUEST_SOURCE_BRANCH_NAME", "feature-branch")
		t.Setenv("CI_MERGE_REQUEST_TARGET_BRANCH_NAME", "main")
		t.Setenv("CI_PIPELINE_ID", "999")
		t.Setenv("CI_COMMIT_SHA", "abc123def456")

		ctx := DetectContext()
		if !ctx.InMR {
			t.Fatal("expected InMR to be true")
		}
		if ctx.ProjectID != "12345" {
			t.Fatalf("ProjectID = %q", ctx.ProjectID)
		}
		if ctx.MRIID != 42 {
			t.Fatalf("MRIID = %d", ctx.MRIID)
		}
	})
}
