package gitlabci

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/edelwud/terraci/pkg/ci"
)

func TestMRService_IsEnabled(t *testing.T) {
	t.Run("not in MR", func(t *testing.T) {
		svc := &MRService{
			context: &MRContext{InMR: false},
			client:  NewClient("", "token"),
		}
		if svc.IsEnabled() {
			t.Error("expected IsEnabled to be false when not in MR")
		}
	})

	t.Run("in MR without token", func(t *testing.T) {
		svc := &MRService{
			context: &MRContext{InMR: true},
			client:  NewClient("", ""),
		}
		if svc.IsEnabled() {
			t.Error("expected IsEnabled to be false without token")
		}
	})

	t.Run("in MR with token, default config", func(t *testing.T) {
		svc := &MRService{
			context: &MRContext{InMR: true},
			client:  NewClient("", "token"),
			config:  nil,
		}
		if !svc.IsEnabled() {
			t.Error("expected IsEnabled to be true by default")
		}
	})

	t.Run("explicitly disabled", func(t *testing.T) {
		enabled := false
		svc := &MRService{
			context: &MRContext{InMR: true},
			client:  NewClient("", "token"),
			config: &MRConfig{
				Comment: &MRCommentConfig{
					Enabled: &enabled,
				},
			},
		}
		if svc.IsEnabled() {
			t.Error("expected IsEnabled to be false when explicitly disabled")
		}
	})
}

func setupMockGitLabServer(t *testing.T, notes string, createCalled, updateCalled *bool) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/notes") && r.Method == "GET":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, notes)
		case strings.Contains(r.URL.Path, "/notes") && r.Method == "POST":
			*createCalled = true
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"id": 1, "body": "test"}`)
		case strings.Contains(r.URL.Path, "/notes/") && r.Method == "PUT":
			*updateCalled = true
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"id": 1, "body": "updated"}`)
		default:
			w.WriteHeader(404)
		}
	}))
}

func TestMRService_UpsertComment_Disabled(t *testing.T) {
	svc := &MRService{
		context: &MRContext{InMR: false},
		client:  NewClient("", ""),
		config:  nil,
	}

	plans := []ci.ModulePlan{{ModuleID: "test/prod/vpc", Status: ci.PlanStatusChanges}}
	err := svc.UpsertComment(plans, nil)
	if err != nil {
		t.Errorf("expected nil error for disabled service, got: %v", err)
	}
}

func TestMRService_UpsertComment_CreateNew(t *testing.T) {
	var createCalled, updateCalled bool
	server := setupMockGitLabServer(t, `[]`, &createCalled, &updateCalled)
	defer server.Close()

	client := NewClient(server.URL, "test-token")

	svc := &MRService{
		context: &MRContext{
			InMR:        true,
			ProjectID:   "123",
			ProjectPath: "group/project",
			MRIID:       1,
			CommitSHA:   "abc123",
			PipelineID:  "456",
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
		t.Error("expected CreateMRNote to be called")
	}
	if updateCalled {
		t.Error("did not expect UpdateMRNote to be called")
	}
}

func TestMRService_UpsertComment_UpdateExisting(t *testing.T) {
	var createCalled, updateCalled bool
	existingNotes := fmt.Sprintf(`[{"id": 42, "body": "old comment %s"}]`, ci.CommentMarker)
	server := setupMockGitLabServer(t, existingNotes, &createCalled, &updateCalled)
	defer server.Close()

	client := NewClient(server.URL, "test-token")

	svc := &MRService{
		context: &MRContext{
			InMR:        true,
			ProjectID:   "123",
			ProjectPath: "group/project",
			MRIID:       1,
			CommitSHA:   "abc123",
			PipelineID:  "456",
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
		t.Error("did not expect CreateMRNote to be called")
	}
	if !updateCalled {
		t.Error("expected UpdateMRNote to be called")
	}
}

func TestMRService_UpsertComment_GetNotesError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	svc := &MRService{
		context:  &MRContext{InMR: true, ProjectID: "123", MRIID: 1},
		client:   client,
		renderer: ci.NewCommentRenderer(),
		config:   nil,
	}

	plans := []ci.ModulePlan{{ModuleID: "test", Status: ci.PlanStatusChanges}}
	err := svc.UpsertComment(plans, nil)
	if err == nil {
		t.Error("expected error when GetMRNotes fails")
	}
}

func TestMRService_UpsertComment_CreateError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/notes") && r.Method == "GET":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `[]`)
		case strings.Contains(r.URL.Path, "/notes") && r.Method == "POST":
			w.WriteHeader(http.StatusInternalServerError)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	svc := &MRService{
		context:  &MRContext{InMR: true, ProjectID: "123", MRIID: 1},
		client:   client,
		renderer: ci.NewCommentRenderer(),
		config:   nil,
	}

	plans := []ci.ModulePlan{{ModuleID: "test", Status: ci.PlanStatusChanges}}
	err := svc.UpsertComment(plans, nil)
	if err == nil {
		t.Error("expected error when CreateMRNote fails")
	}
}

func TestMRService_UpsertComment_OnChangesOnly_NoChanges(t *testing.T) {
	var createCalled, updateCalled bool
	server := setupMockGitLabServer(t, `[]`, &createCalled, &updateCalled)
	defer server.Close()

	client := NewClient(server.URL, "test-token")

	svc := &MRService{
		context: &MRContext{
			InMR:      true,
			ProjectID: "123",
			MRIID:     1,
		},
		client:   client,
		renderer: ci.NewCommentRenderer(),
		config: &MRConfig{
			Comment: &MRCommentConfig{
				OnChangesOnly: true,
			},
		},
	}

	// Plans with no changes — should skip comment
	plans := []ci.ModulePlan{{ModuleID: "test/prod/vpc", Status: ci.PlanStatusNoChanges}}
	err := svc.UpsertComment(plans, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if createCalled {
		t.Error("did not expect CreateMRNote to be called when on_changes_only and no changes")
	}
	if updateCalled {
		t.Error("did not expect UpdateMRNote to be called when on_changes_only and no changes")
	}
}
