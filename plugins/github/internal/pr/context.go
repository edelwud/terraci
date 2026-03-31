package pr

import (
	"encoding/json"
	"os"
	"strconv"
	"strings"
)

type Context struct {
	Owner        string
	Repo         string
	PRNumber     int
	SourceBranch string
	TargetBranch string
	RunID        string
	CommitSHA    string
	InPR         bool
}

func DetectContext() *Context {
	repository := os.Getenv("GITHUB_REPOSITORY")
	owner, repo := ParseRepository(repository)

	ctx := &Context{
		Owner:     owner,
		Repo:      repo,
		RunID:     os.Getenv("GITHUB_RUN_ID"),
		CommitSHA: os.Getenv("GITHUB_SHA"),
	}

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

	if !ctx.InPR {
		ctx.PRNumber = GetPRNumberFromEvent()
		ctx.InPR = ctx.PRNumber > 0
	}

	ctx.SourceBranch = os.Getenv("GITHUB_HEAD_REF")
	ctx.TargetBranch = os.Getenv("GITHUB_BASE_REF")
	return ctx
}

func GetPRNumberFromEvent() int {
	eventPath := os.Getenv("GITHUB_EVENT_PATH")
	if eventPath == "" {
		return 0
	}

	data, err := os.ReadFile(eventPath) //nolint:gosec // path provided by GitHub Actions runtime
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
