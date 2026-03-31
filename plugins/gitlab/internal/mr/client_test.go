package mr

import (
	"testing"
)

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

func TestNewClientFromEnv(t *testing.T) {
	t.Run("with GITLAB_TOKEN", func(t *testing.T) {
		t.Setenv("CI_SERVER_URL", "https://gitlab.example.com")
		t.Setenv("GITLAB_TOKEN", "glpat-xxx")
		t.Setenv("CI_JOB_TOKEN", "")
		client := NewClientFromEnv()
		if !client.HasToken() {
			t.Error("expected HasToken true")
		}
	})

	t.Run("falls back to CI_JOB_TOKEN", func(t *testing.T) {
		t.Setenv("CI_SERVER_URL", "")
		t.Setenv("GITLAB_TOKEN", "")
		t.Setenv("CI_JOB_TOKEN", "job-token")
		client := NewClientFromEnv()
		if !client.HasToken() {
			t.Error("expected HasToken true")
		}
	})

	t.Run("no tokens", func(t *testing.T) {
		t.Setenv("CI_SERVER_URL", "")
		t.Setenv("GITLAB_TOKEN", "")
		t.Setenv("CI_JOB_TOKEN", "")
		client := NewClientFromEnv()
		if client.HasToken() {
			t.Error("expected HasToken false")
		}
	})
}
