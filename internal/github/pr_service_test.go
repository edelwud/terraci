package github

import (
	"testing"

	"github.com/edelwud/terraci/pkg/config"
)

func boolPtr(b bool) *bool { return &b }

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
