// Package gitlab provides GitLab API client for MR integration
package gitlab

import (
	"os"
	"strconv"
	"strings"

	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// Client wraps the official GitLab client
type Client struct {
	client *gitlab.Client
	token  string
}

// MRContext contains information about the current MR context
type MRContext struct {
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

// NewClient creates a new GitLab API client
func NewClient(baseURL, token string) *Client {
	var client *gitlab.Client
	var err error

	if baseURL != "" {
		baseURL = strings.TrimSuffix(baseURL, "/")
		client, err = gitlab.NewClient(token, gitlab.WithBaseURL(baseURL))
	} else {
		client, err = gitlab.NewClient(token)
	}

	if err != nil {
		// Return client without underlying gitlab client - HasToken will return false
		return &Client{token: token}
	}

	return &Client{
		client: client,
		token:  token,
	}
}

// NewClientFromEnv creates a client from GitLab CI environment variables
func NewClientFromEnv() *Client {
	baseURL := os.Getenv("CI_SERVER_URL")

	token := os.Getenv("GITLAB_TOKEN")
	if token == "" {
		token = os.Getenv("CI_JOB_TOKEN")
	}

	return NewClient(baseURL, token)
}

// DetectMRContext detects if we're running in a GitLab MR pipeline
func DetectMRContext() *MRContext {
	ctx := &MRContext{
		ProjectID:    os.Getenv("CI_PROJECT_ID"),
		ProjectPath:  os.Getenv("CI_PROJECT_PATH"),
		SourceBranch: os.Getenv("CI_MERGE_REQUEST_SOURCE_BRANCH_NAME"),
		TargetBranch: os.Getenv("CI_MERGE_REQUEST_TARGET_BRANCH_NAME"),
		PipelineID:   os.Getenv("CI_PIPELINE_ID"),
		JobID:        os.Getenv("CI_JOB_ID"),
		CommitSHA:    os.Getenv("CI_COMMIT_SHA"),
	}

	// Check for MR IID
	mrIIDStr := os.Getenv("CI_MERGE_REQUEST_IID")
	if mrIIDStr != "" {
		if iid, err := strconv.ParseInt(mrIIDStr, 10, 64); err == nil {
			ctx.MRIID = iid
			ctx.InMR = true
		}
	}

	return ctx
}

// HasToken returns true if a token is configured
func (c *Client) HasToken() bool {
	return c.token != "" && c.client != nil
}

// BaseURL returns the GitLab instance base URL
func (c *Client) BaseURL() string {
	if c.client == nil {
		return ""
	}
	return strings.TrimSuffix(c.client.BaseURL().String(), "/api/v4/")
}

// GetMRNotes retrieves all notes for an MR
func (c *Client) GetMRNotes(projectID string, mrIID int64) ([]*gitlab.Note, error) {
	opts := &gitlab.ListMergeRequestNotesOptions{
		ListOptions: gitlab.ListOptions{PerPage: 100},
	}

	var allNotes []*gitlab.Note
	for {
		notes, resp, err := c.client.Notes.ListMergeRequestNotes(projectID, mrIID, opts)
		if err != nil {
			return nil, err
		}

		allNotes = append(allNotes, notes...)

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return allNotes, nil
}

// CreateMRNote creates a new note on an MR
func (c *Client) CreateMRNote(projectID string, mrIID int64, body string) (*gitlab.Note, error) {
	opts := &gitlab.CreateMergeRequestNoteOptions{
		Body: gitlab.Ptr(body),
	}

	note, _, err := c.client.Notes.CreateMergeRequestNote(projectID, mrIID, opts)
	return note, err
}

// UpdateMRNote updates an existing note on an MR
func (c *Client) UpdateMRNote(projectID string, mrIID, noteID int64, body string) (*gitlab.Note, error) {
	opts := &gitlab.UpdateMergeRequestNoteOptions{
		Body: gitlab.Ptr(body),
	}

	note, _, err := c.client.Notes.UpdateMergeRequestNote(projectID, mrIID, noteID, opts)
	return note, err
}

// AddMRLabels adds labels to an MR
func (c *Client) AddMRLabels(projectID string, mrIID int64, labels []string) error {
	labelsArg := gitlab.LabelOptions(labels)
	opts := &gitlab.UpdateMergeRequestOptions{
		AddLabels: &labelsArg,
	}

	_, _, err := c.client.MergeRequests.UpdateMergeRequest(projectID, mrIID, opts)
	return err
}
