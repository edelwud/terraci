package mr

import (
	"fmt"
	"strings"
	"testing"

	gitlab "gitlab.com/gitlab-org/api/client-go"

	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/ci/citest"
)

func TestMRService_IsEnabled(t *testing.T) {
	citest.RunEnabledCases(t, []citest.EnabledCase[*Context, struct{}]{
		{Name: "not in MR", Context: &Context{InMR: false}, HasToken: true, Expected: false},
		{Name: "in MR without token", Context: &Context{InMR: true}, HasToken: false, Expected: false},
		{Name: "in MR with token", Context: &Context{InMR: true}, HasToken: true, Expected: true},
	}, func(t *testing.T, ctx *Context, _ struct{}, hasToken bool) bool {
		return newServiceScenario(t).withContext(ctx).withToken(hasToken).service.IsEnabled()
	})
}

func TestMRService_UpsertComment_Disabled(t *testing.T) {
	err := newServiceScenario(t).
		withContext(&Context{InMR: false}).
		withToken(false).
		upsert("test body")
	if err != nil {
		t.Errorf("expected nil error for disabled service, got: %v", err)
	}
}

func TestMRService_UpsertComment_CreateNew(t *testing.T) {
	scenario := newServiceScenario(t)

	err := scenario.upsert(ci.CommentMarker + "\n\n## Test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	citest.AssertCreateOnly(t, scenario.client.createdBody, scenario.client.updatedBody)
}

func TestMRService_UpsertComment_UpdateExisting(t *testing.T) {
	scenario := newServiceScenario(t).withNotes(
		&gitlab.Note{ID: 42, Body: "old comment " + ci.CommentMarker},
	)

	err := scenario.upsert(ci.CommentMarker + "\n\n## Test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	citest.AssertUpdateOnly(t, scenario.client.createdBody, scenario.client.updatedBody, scenario.client.updatedNoteID, 42)
}

func TestMRService_UpsertComment_GetNotesError(t *testing.T) {
	err := newServiceScenario(t).
		withGetError(fmt.Errorf("boom")).
		upsert("test body")
	if err == nil {
		t.Error("expected error when GetMRNotes fails")
	}
}

func TestMRService_UpsertComment_CreateError(t *testing.T) {
	err := newServiceScenario(t).
		withCreateError(fmt.Errorf("boom")).
		upsert("test body")
	if err == nil {
		t.Error("expected error when CreateMRNote fails")
	}
}

func TestMRService_UpsertComment_UpdateError(t *testing.T) {
	err := newServiceScenario(t).
		withNotes(&gitlab.Note{ID: 7, Body: ci.CommentMarker + " existing"}).
		withUpdateError(fmt.Errorf("boom")).
		upsert("test body")
	if err == nil {
		t.Error("expected error when UpdateMRNote fails")
	}
}

func TestMRService_CurrentCommentBody(t *testing.T) {
	body := ci.EmbedManagedLabels(ci.CommentMarker+"\n\n## Test", []string{"terraform"})
	got, found, err := newServiceScenario(t).
		withNotes(&gitlab.Note{ID: 7, Body: body}).
		service.CurrentCommentBody(t.Context())
	if err != nil {
		t.Fatalf("CurrentCommentBody() error = %v", err)
	}
	if !found {
		t.Fatal("CurrentCommentBody() found = false, want true")
	}
	if got != body {
		t.Fatalf("CurrentCommentBody() body = %q, want %q", got, body)
	}
}

func TestMRService_SyncLabels_AddsAndRemovesManagedDiff(t *testing.T) {
	scenario := newServiceScenario(t)

	err := scenario.service.SyncLabels(t.Context(), []string{"keep", "stale"}, []string{"keep", "terraform"})
	if err != nil {
		t.Fatalf("SyncLabels() error = %v", err)
	}

	if got := strings.Join(scenario.client.removedLabels, ","); got != "stale" {
		t.Fatalf("removed labels = %v, want [stale]", scenario.client.removedLabels)
	}
	if got := strings.Join(scenario.client.addedLabels, ","); got != "terraform" {
		t.Fatalf("added labels = %v, want [terraform]", scenario.client.addedLabels)
	}
}
