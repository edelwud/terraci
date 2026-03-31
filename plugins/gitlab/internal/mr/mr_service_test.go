package mr

import (
	"fmt"
	"testing"

	gitlab "gitlab.com/gitlab-org/api/client-go"

	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/ciprovider/ciprovidertest"
	configpkg "github.com/edelwud/terraci/plugins/gitlab/internal/config"
)

func TestMRService_IsEnabled(t *testing.T) {
	ciprovidertest.RunEnabledCases(t, []ciprovidertest.EnabledCase[*Context, *configpkg.MRConfig]{
		{Name: "not in MR", Context: &Context{InMR: false}, HasToken: true, Expected: false},
		{Name: "in MR without token", Context: &Context{InMR: true}, HasToken: false, Expected: false},
		{Name: "in MR with token, default config", Context: &Context{InMR: true}, HasToken: true, Expected: true},
		{
			Name:     "explicitly disabled",
			Context:  &Context{InMR: true},
			HasToken: true,
			Config: &configpkg.MRConfig{
				Comment: &configpkg.MRCommentConfig{Enabled: ciprovidertest.BoolPtr(false)},
			},
			Expected: false,
		},
	}, func(t *testing.T, ctx *Context, cfg *configpkg.MRConfig, hasToken bool) bool {
		return newServiceScenario(t).withContext(ctx).withConfig(cfg).withToken(hasToken).service.IsEnabled()
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
	ciprovidertest.AssertCreateOnly(t, scenario.client.createdBody, scenario.client.updatedBody)
}

func TestMRService_UpsertComment_UpdateExisting(t *testing.T) {
	scenario := newServiceScenario(t).withNotes(
		&gitlab.Note{ID: 42, Body: "old comment " + ci.CommentMarker},
	)

	err := scenario.upsert(ci.CommentMarker + "\n\n## Test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ciprovidertest.AssertUpdateOnly(t, scenario.client.createdBody, scenario.client.updatedBody, scenario.client.updatedNoteID, 42)
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

func TestMRService_UpsertComment_OnChangesOnly_NoChanges(t *testing.T) {
	scenario := newServiceScenario(t).withConfig(&configpkg.MRConfig{
		Comment: &configpkg.MRCommentConfig{
			OnChangesOnly: true,
		},
	})

	err := scenario.upsert("test body")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if scenario.client.createdBody == "" {
		t.Error("expected CreateMRNote to be called")
	}
}
