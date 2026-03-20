package github

import (
	"testing"
)

func TestParseRepository(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantOwner string
		wantRepo  string
	}{
		{
			name:      "valid owner/repo",
			input:     "owner/repo",
			wantOwner: "owner",
			wantRepo:  "repo",
		},
		{
			name:      "empty string",
			input:     "",
			wantOwner: "",
			wantRepo:  "",
		},
		{
			name:      "no slash",
			input:     "noslash",
			wantOwner: "",
			wantRepo:  "",
		},
		{
			name:      "multiple slashes uses SplitN limit 2",
			input:     "a/b/c",
			wantOwner: "a",
			wantRepo:  "b/c",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo := parseRepository(tt.input)
			if owner != tt.wantOwner {
				t.Errorf("parseRepository(%q) owner = %q, want %q", tt.input, owner, tt.wantOwner)
			}
			if repo != tt.wantRepo {
				t.Errorf("parseRepository(%q) repo = %q, want %q", tt.input, repo, tt.wantRepo)
			}
		})
	}
}

func TestNewClient_WithToken(t *testing.T) {
	c := NewClient("token123", "owner/repo")
	if !c.HasToken() {
		t.Error("expected HasToken() to be true when token is provided")
	}
}

func TestNewClient_WithoutToken(t *testing.T) {
	c := NewClient("", "owner/repo")
	if c.HasToken() {
		t.Error("expected HasToken() to be false when token is empty")
	}
}

func TestNewClientFromEnv(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "env-token")
	t.Setenv("GITHUB_REPOSITORY", "envowner/envrepo")

	c := NewClientFromEnv()
	if !c.HasToken() {
		t.Error("expected HasToken() to be true when GITHUB_TOKEN is set")
	}
	if c.owner != "envowner" {
		t.Errorf("expected owner %q, got %q", "envowner", c.owner)
	}
	if c.repo != "envrepo" {
		t.Errorf("expected repo %q, got %q", "envrepo", c.repo)
	}
}
