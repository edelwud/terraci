package github

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	gh "github.com/google/go-github/v68/github"

	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/config"
)

func boolPtr(b bool) *bool { return &b }

func setupMockGitHubServer(t *testing.T, comments string, createCalled, updateCalled *bool) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/comments") && r.Method == "GET":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, comments)
		case strings.HasSuffix(r.URL.Path, "/comments") && r.Method == "POST":
			*createCalled = true
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"id": 1, "body": "test"}`)
		case strings.Contains(r.URL.Path, "/comments/") && r.Method == "PATCH":
			*updateCalled = true
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"id": 1, "body": "updated"}`)
		default:
			w.WriteHeader(404)
		}
	}))
}

func newMockGitHubClient(t *testing.T, serverURL string) *Client {
	t.Helper()
	ghClient := gh.NewClient(nil).WithAuthToken("test-token")
	baseURL, err := url.Parse(serverURL + "/")
	if err != nil {
		t.Fatalf("failed to parse server URL: %v", err)
	}
	ghClient.BaseURL = baseURL

	return &Client{
		client: ghClient,
		owner:  "owner",
		repo:   "repo",
		token:  "test-token",
	}
}

func TestPRService_IsEnabled_NoPR(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "tok")
	t.Setenv("GITHUB_REPOSITORY", "owner/repo")
	t.Setenv("GITHUB_REF", "refs/heads/main")
	t.Setenv("GITHUB_EVENT_PATH", "")

	svc := NewPRServiceFromEnv(nil)

	if svc.IsEnabled() {
		t.Error("expected IsEnabled() to be false when not in a PR")
	}
}

func TestPRService_IsEnabled_NoToken(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "")
	t.Setenv("GITHUB_REPOSITORY", "owner/repo")
	t.Setenv("GITHUB_REF", "refs/pull/1/merge")
	t.Setenv("GITHUB_EVENT_PATH", "")

	svc := NewPRServiceFromEnv(nil)

	if svc.IsEnabled() {
		t.Error("expected IsEnabled() to be false when no token is set")
	}
}

func TestPRService_IsEnabled_Defaults(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "tok")
	t.Setenv("GITHUB_REPOSITORY", "o/r")
	t.Setenv("GITHUB_REF", "refs/pull/1/merge")
	t.Setenv("GITHUB_EVENT_PATH", "")

	svc := NewPRServiceFromEnv(nil)

	if !svc.IsEnabled() {
		t.Error("expected IsEnabled() to be true with token, PR ref, and nil config")
	}
}

func TestPRService_IsEnabled_Disabled(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "tok")
	t.Setenv("GITHUB_REPOSITORY", "o/r")
	t.Setenv("GITHUB_REF", "refs/pull/1/merge")
	t.Setenv("GITHUB_EVENT_PATH", "")

	cfg := &config.PRConfig{
		Comment: &config.MRCommentConfig{
			Enabled: boolPtr(false),
		},
	}

	svc := NewPRServiceFromEnv(cfg)

	if svc.IsEnabled() {
		t.Error("expected IsEnabled() to be false when config has Enabled=false")
	}
}

func TestPRService_UpsertComment_Disabled(t *testing.T) {
	svc := &PRService{
		context: &PRContext{InPR: false},
		client:  &Client{token: ""},
		config:  nil,
	}

	plans := []ci.ModulePlan{{ModuleID: "test/prod/vpc", Status: ci.PlanStatusChanges}}
	err := svc.UpsertComment(plans, nil)
	if err != nil {
		t.Errorf("expected nil error for disabled service, got: %v", err)
	}
}

func TestPRService_UpsertComment_CreateNew(t *testing.T) {
	var createCalled, updateCalled bool
	server := setupMockGitHubServer(t, `[]`, &createCalled, &updateCalled)
	defer server.Close()

	client := newMockGitHubClient(t, server.URL)

	svc := &PRService{
		context: &PRContext{
			InPR:      true,
			Owner:     "owner",
			Repo:      "repo",
			PRNumber:  1,
			RunID:     "789",
			CommitSHA: "abc123",
		},
		client:   client,
		renderer: ci.NewCommentRenderer(),
		config:   nil,
	}

	plans := []ci.ModulePlan{{ModuleID: "test/prod/vpc", Status: ci.PlanStatusChanges}}
	err := svc.UpsertComment(plans, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !createCalled {
		t.Error("expected CreateIssueComment to be called")
	}
	if updateCalled {
		t.Error("did not expect UpdateIssueComment to be called")
	}
}

func TestPRService_UpsertComment_UpdateExisting(t *testing.T) {
	var createCalled, updateCalled bool
	existingComments := fmt.Sprintf(`[{"id": 42, "body": "old comment %s"}]`, ci.CommentMarker)
	server := setupMockGitHubServer(t, existingComments, &createCalled, &updateCalled)
	defer server.Close()

	client := newMockGitHubClient(t, server.URL)

	svc := &PRService{
		context: &PRContext{
			InPR:      true,
			Owner:     "owner",
			Repo:      "repo",
			PRNumber:  1,
			RunID:     "789",
			CommitSHA: "abc123",
		},
		client:   client,
		renderer: ci.NewCommentRenderer(),
		config:   nil,
	}

	plans := []ci.ModulePlan{{ModuleID: "test/prod/vpc", Status: ci.PlanStatusChanges}}
	err := svc.UpsertComment(plans, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if createCalled {
		t.Error("did not expect CreateIssueComment to be called")
	}
	if !updateCalled {
		t.Error("expected UpdateIssueComment to be called")
	}
}

func TestPRService_UpsertComment_ListError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := newMockGitHubClient(t, server.URL)

	svc := &PRService{
		context: &PRContext{
			InPR:     true,
			Owner:    "owner",
			Repo:     "repo",
			PRNumber: 1,
		},
		client:   client,
		renderer: ci.NewCommentRenderer(),
		config:   nil,
	}

	plans := []ci.ModulePlan{{ModuleID: "test", Status: ci.PlanStatusChanges}}
	err := svc.UpsertComment(plans, nil)
	if err == nil {
		t.Error("expected error when ListIssueComments fails")
	}
}

func TestPRService_UpsertComment_CreateError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/comments") && r.Method == "GET":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `[]`)
		case strings.HasSuffix(r.URL.Path, "/comments") && r.Method == "POST":
			w.WriteHeader(http.StatusInternalServerError)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := newMockGitHubClient(t, server.URL)

	svc := &PRService{
		context: &PRContext{
			InPR:     true,
			Owner:    "owner",
			Repo:     "repo",
			PRNumber: 1,
		},
		client:   client,
		renderer: ci.NewCommentRenderer(),
		config:   nil,
	}

	plans := []ci.ModulePlan{{ModuleID: "test", Status: ci.PlanStatusChanges}}
	err := svc.UpsertComment(plans, nil)
	if err == nil {
		t.Error("expected error when CreateIssueComment fails")
	}
}

func TestPRService_UpsertComment_OnChangesOnly_NoChanges(t *testing.T) {
	var createCalled, updateCalled bool
	server := setupMockGitHubServer(t, `[]`, &createCalled, &updateCalled)
	defer server.Close()

	client := newMockGitHubClient(t, server.URL)

	svc := &PRService{
		context: &PRContext{
			InPR:     true,
			Owner:    "owner",
			Repo:     "repo",
			PRNumber: 1,
		},
		client:   client,
		renderer: ci.NewCommentRenderer(),
		config: &config.PRConfig{
			Comment: &config.MRCommentConfig{
				OnChangesOnly: true,
			},
		},
	}

	plans := []ci.ModulePlan{{ModuleID: "test/prod/vpc", Status: ci.PlanStatusNoChanges}}
	err := svc.UpsertComment(plans, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if createCalled {
		t.Error("did not expect CreateIssueComment to be called when on_changes_only and no changes")
	}
	if updateCalled {
		t.Error("did not expect UpdateIssueComment to be called when on_changes_only and no changes")
	}
}
