package github

import (
	"encoding/json"
	"os"
	"strconv"
	"strings"
)

// PRContext contains information about the current PR context
type PRContext struct {
	Owner        string
	Repo         string
	PRNumber     int
	SourceBranch string
	TargetBranch string
	RunID        string
	CommitSHA    string
	InPR         bool
}

// DetectPRContext detects if we're running in a GitHub Actions PR workflow
func DetectPRContext() *PRContext {
	repository := os.Getenv("GITHUB_REPOSITORY")
	owner, repo := parseRepository(repository)

	ctx := &PRContext{
		Owner:     owner,
		Repo:      repo,
		RunID:     os.Getenv("GITHUB_RUN_ID"),
		CommitSHA: os.Getenv("GITHUB_SHA"),
	}

	// Try to get PR number from GITHUB_REF (refs/pull/123/merge)
	ref := os.Getenv("GITHUB_REF")
	if strings.HasPrefix(ref, "refs/pull/") {
		parts := strings.Split(ref, "/")
		const minRefParts = 3
		if len(parts) >= minRefParts {
			if num, err := strconv.Atoi(parts[2]); err == nil {
				ctx.PRNumber = num
				ctx.InPR = true
			}
		}
	}

	// Also try to get PR number from event payload
	if !ctx.InPR {
		ctx.PRNumber = getPRNumberFromEvent()
		ctx.InPR = ctx.PRNumber > 0
	}

	// Get branch info
	ctx.SourceBranch = os.Getenv("GITHUB_HEAD_REF")
	ctx.TargetBranch = os.Getenv("GITHUB_BASE_REF")

	return ctx
}

// getPRNumberFromEvent reads the PR number from the GitHub event payload
func getPRNumberFromEvent() int {
	eventPath := os.Getenv("GITHUB_EVENT_PATH")
	if eventPath == "" {
		return 0
	}

	data, err := os.ReadFile(eventPath)
	if err != nil {
		return 0
	}

	var event struct {
		PullRequest *struct {
			Number int `json:"number"`
		} `json:"pull_request"`
	}

	if err := json.Unmarshal(data, &event); err != nil {
		return 0
	}

	if event.PullRequest != nil {
		return event.PullRequest.Number
	}

	return 0
}
