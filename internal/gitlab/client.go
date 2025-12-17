// Package gitlab provides GitLab API client for MR integration
package gitlab

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	// DefaultGitLabURL is the default GitLab instance URL
	DefaultGitLabURL = "https://gitlab.com"
	// DefaultHTTPTimeout is the default HTTP client timeout
	DefaultHTTPTimeout = 30 * time.Second
)

// Client provides GitLab API operations
type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
	projectID  string
	mrIID      int
}

// MRContext contains information about the current MR context
type MRContext struct {
	ProjectID    string
	ProjectPath  string
	MRIID        int
	SourceBranch string
	TargetBranch string
	PipelineID   string
	JobID        string
	CommitSHA    string
	InMR         bool
}

// NewClient creates a new GitLab API client
func NewClient(baseURL, token string) *Client {
	if baseURL == "" {
		baseURL = DefaultGitLabURL
	}
	baseURL = strings.TrimSuffix(baseURL, "/")

	return &Client{
		baseURL: baseURL,
		token:   token,
		httpClient: &http.Client{
			Timeout: DefaultHTTPTimeout,
		},
	}
}

// NewClientFromEnv creates a client from GitLab CI environment variables
func NewClientFromEnv() *Client {
	baseURL := os.Getenv("CI_SERVER_URL")
	if baseURL == "" {
		baseURL = DefaultGitLabURL
	}

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
		if iid, err := strconv.Atoi(mrIIDStr); err == nil {
			ctx.MRIID = iid
			ctx.InMR = true
		}
	}

	return ctx
}

// SetProject sets the project context for API calls
func (c *Client) SetProject(projectID string) {
	c.projectID = projectID
}

// SetMR sets the MR context for API calls
func (c *Client) SetMR(mrIID int) {
	c.mrIID = mrIID
}

// HasToken returns true if a token is configured
func (c *Client) HasToken() bool {
	return c.token != ""
}

// Note represents a GitLab MR note (comment)
type Note struct {
	ID        int       `json:"id"`
	Body      string    `json:"body"`
	Author    Author    `json:"author"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	System    bool      `json:"system"`
}

// Author represents a GitLab user
type Author struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Name     string `json:"name"`
}

// Label represents a GitLab label
type Label struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

// GetMRNotes retrieves all notes for an MR
func (c *Client) GetMRNotes(projectID string, mrIID int) ([]Note, error) {
	endpoint := fmt.Sprintf("/api/v4/projects/%s/merge_requests/%d/notes",
		url.PathEscape(projectID), mrIID)

	var allNotes []Note
	page := 1

	for {
		req, err := c.newRequest(context.Background(), "GET", endpoint+"?page="+strconv.Itoa(page)+"&per_page=100", nil)
		if err != nil {
			return nil, err
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to get MR notes: %w", err)
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to read response: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("GitLab API error: %s - %s", resp.Status, string(body))
		}

		var notes []Note
		if err := json.Unmarshal(body, &notes); err != nil {
			return nil, fmt.Errorf("failed to parse notes: %w", err)
		}

		if len(notes) == 0 {
			break
		}

		allNotes = append(allNotes, notes...)
		page++
	}

	return allNotes, nil
}

// CreateMRNote creates a new note on an MR
func (c *Client) CreateMRNote(projectID string, mrIID int, body string) (*Note, error) {
	endpoint := fmt.Sprintf("/api/v4/projects/%s/merge_requests/%d/notes",
		url.PathEscape(projectID), mrIID)

	payload := map[string]string{"body": body}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := c.newRequest(context.Background(), "POST", endpoint, bytes.NewReader(jsonData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to create note: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("GitLab API error: %s - %s", resp.Status, string(respBody))
	}

	var note Note
	if err := json.Unmarshal(respBody, &note); err != nil {
		return nil, fmt.Errorf("failed to parse note: %w", err)
	}

	return &note, nil
}

// UpdateMRNote updates an existing note on an MR
func (c *Client) UpdateMRNote(projectID string, mrIID, noteID int, body string) (*Note, error) {
	endpoint := fmt.Sprintf("/api/v4/projects/%s/merge_requests/%d/notes/%d",
		url.PathEscape(projectID), mrIID, noteID)

	payload := map[string]string{"body": body}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := c.newRequest(context.Background(), "PUT", endpoint, bytes.NewReader(jsonData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to update note: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitLab API error: %s - %s", resp.Status, string(respBody))
	}

	var note Note
	if err := json.Unmarshal(respBody, &note); err != nil {
		return nil, fmt.Errorf("failed to parse note: %w", err)
	}

	return &note, nil
}

// AddMRLabels adds labels to an MR
func (c *Client) AddMRLabels(projectID string, mrIID int, labels []string) error {
	endpoint := fmt.Sprintf("/api/v4/projects/%s/merge_requests/%d",
		url.PathEscape(projectID), mrIID)

	payload := map[string]string{
		"add_labels": strings.Join(labels, ","),
	}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := c.newRequest(context.Background(), "PUT", endpoint, bytes.NewReader(jsonData))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to add labels: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("GitLab API error: %s", resp.Status)
		}
		return fmt.Errorf("GitLab API error: %s - %s", resp.Status, string(body))
	}

	return nil
}

func (c *Client) newRequest(ctx context.Context, method, endpoint string, body io.Reader) (*http.Request, error) {
	reqURL := c.baseURL + endpoint
	req, err := http.NewRequestWithContext(ctx, method, reqURL, body)
	if err != nil {
		return nil, err
	}

	if c.token != "" {
		req.Header.Set("PRIVATE-TOKEN", c.token)
	}

	return req, nil
}
