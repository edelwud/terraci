package mr

import (
	"os"
	"strconv"
)

// Context contains information about the current MR context.
type Context struct {
	ProjectID    string
	ProjectPath  string
	MRIID        int64
	SourceBranch string
	TargetBranch string
	PipelineID   string
	JobID        string
	CommitSHA    string
	InMR         bool
}

// DetectContext detects if we're running in a GitLab MR pipeline.
func DetectContext() *Context {
	ctx := &Context{
		ProjectID:    os.Getenv("CI_PROJECT_ID"),
		ProjectPath:  os.Getenv("CI_PROJECT_PATH"),
		SourceBranch: os.Getenv("CI_MERGE_REQUEST_SOURCE_BRANCH_NAME"),
		TargetBranch: os.Getenv("CI_MERGE_REQUEST_TARGET_BRANCH_NAME"),
		PipelineID:   os.Getenv("CI_PIPELINE_ID"),
		JobID:        os.Getenv("CI_JOB_ID"),
		CommitSHA:    os.Getenv("CI_COMMIT_SHA"),
	}

	mrIIDStr := os.Getenv("CI_MERGE_REQUEST_IID")
	if mrIIDStr != "" {
		if iid, err := strconv.ParseInt(mrIIDStr, 10, 64); err == nil {
			ctx.MRIID = iid
			ctx.InMR = true
		}
	}

	return ctx
}
