package gitlab

import (
	"os"
	"testing"
)

func TestDetectMRContext(t *testing.T) {
	// Save original env vars
	origVars := map[string]string{
		"CI_PROJECT_ID":                       os.Getenv("CI_PROJECT_ID"),
		"CI_PROJECT_PATH":                     os.Getenv("CI_PROJECT_PATH"),
		"CI_MERGE_REQUEST_IID":                os.Getenv("CI_MERGE_REQUEST_IID"),
		"CI_MERGE_REQUEST_SOURCE_BRANCH_NAME": os.Getenv("CI_MERGE_REQUEST_SOURCE_BRANCH_NAME"),
		"CI_MERGE_REQUEST_TARGET_BRANCH_NAME": os.Getenv("CI_MERGE_REQUEST_TARGET_BRANCH_NAME"),
		"CI_PIPELINE_ID":                      os.Getenv("CI_PIPELINE_ID"),
		"CI_COMMIT_SHA":                       os.Getenv("CI_COMMIT_SHA"),
	}

	// Restore env vars after test
	defer func() {
		for k, v := range origVars {
			if v == "" {
				os.Unsetenv(k)
			} else {
				os.Setenv(k, v)
			}
		}
	}()

	t.Run("not in MR", func(t *testing.T) {
		os.Unsetenv("CI_MERGE_REQUEST_IID")

		ctx := DetectMRContext()
		if ctx.InMR {
			t.Error("expected InMR to be false")
		}
	})

	t.Run("in MR", func(t *testing.T) {
		os.Setenv("CI_PROJECT_ID", "12345")
		os.Setenv("CI_PROJECT_PATH", "group/project")
		os.Setenv("CI_MERGE_REQUEST_IID", "42")
		os.Setenv("CI_MERGE_REQUEST_SOURCE_BRANCH_NAME", "feature-branch")
		os.Setenv("CI_MERGE_REQUEST_TARGET_BRANCH_NAME", "main")
		os.Setenv("CI_PIPELINE_ID", "999")
		os.Setenv("CI_COMMIT_SHA", "abc123def456")

		ctx := DetectMRContext()

		if !ctx.InMR {
			t.Error("expected InMR to be true")
		}
		if ctx.ProjectID != "12345" {
			t.Errorf("expected ProjectID '12345', got %q", ctx.ProjectID)
		}
		if ctx.MRIID != 42 {
			t.Errorf("expected MRIID 42, got %d", ctx.MRIID)
		}
		if ctx.SourceBranch != "feature-branch" {
			t.Errorf("expected SourceBranch 'feature-branch', got %q", ctx.SourceBranch)
		}
	})
}

func TestNewClient(t *testing.T) {
	t.Run("default URL", func(t *testing.T) {
		client := NewClient("", "token123")
		expected := "https://gitlab.com"
		if client.BaseURL() != expected {
			t.Errorf("expected default URL %q, got %q", expected, client.BaseURL())
		}
		if !client.HasToken() {
			t.Error("expected HasToken to be true")
		}
	})

	t.Run("custom URL", func(t *testing.T) {
		client := NewClient("https://gitlab.example.com/", "token")
		expected := "https://gitlab.example.com"
		if client.BaseURL() != expected {
			t.Errorf("expected URL without trailing slash %q, got %q", expected, client.BaseURL())
		}
	})
}

func TestClient_HasToken(t *testing.T) {
	t.Run("with token", func(t *testing.T) {
		client := NewClient("", "token123")
		if !client.HasToken() {
			t.Error("expected HasToken to be true")
		}
	})

	t.Run("without token", func(t *testing.T) {
		client := NewClient("", "")
		if client.HasToken() {
			t.Error("expected HasToken to be false")
		}
	})
}
