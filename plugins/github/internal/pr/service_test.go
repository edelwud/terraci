package pr

import (
	"fmt"
	"testing"

	gh "github.com/google/go-github/v68/github"

	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/ci/citest"
)

func TestService_IsEnabled(t *testing.T) {
	citest.RunEnabledCases(t, []citest.EnabledCase[*Context, struct{}]{
		{Name: "not in PR", Context: &Context{InPR: false}, HasToken: true, Expected: false},
		{Name: "in PR without token", Context: &Context{InPR: true}, HasToken: false, Expected: false},
		{Name: "in PR with token", Context: &Context{InPR: true}, HasToken: true, Expected: true},
	}, func(t *testing.T, ctx *Context, _ struct{}, hasToken bool) bool {
		return newServiceScenario(t).withContext(ctx).withToken(hasToken).service.IsEnabled()
	})
}

func TestService_UpsertComment_Disabled(t *testing.T) {
	if err := newServiceScenario(t).
		withContext(&Context{InPR: false}).
		withToken(false).
		upsert("test body"); err != nil {
		t.Errorf("expected nil error for disabled service, got: %v", err)
	}
}

func TestService_UpsertComment_CreateNew(t *testing.T) {
	scenario := newServiceScenario(t)

	if err := scenario.upsert(ci.CommentMarker + "\n\n## Test"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	citest.AssertCreateOnly(t, scenario.client.createdBody, scenario.client.updatedBody)
}

func TestService_UpsertComment_UpdateExisting(t *testing.T) {
	scenario := newServiceScenario(t).withComments(
		&gh.IssueComment{ID: gh.Ptr(int64(42)), Body: gh.Ptr("old comment " + ci.CommentMarker)},
	)

	if err := scenario.upsert(ci.CommentMarker + "\n\n## Test"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	citest.AssertUpdateOnly(t, scenario.client.createdBody, scenario.client.updatedBody, scenario.client.updatedCommentID, 42)
}

func TestService_UpsertComment_ListError(t *testing.T) {
	if err := newServiceScenario(t).
		withListError(fmt.Errorf("boom")).
		upsert("test body"); err == nil {
		t.Error("expected error when ListIssueComments fails")
	}
}

func TestService_UpsertComment_CreateError(t *testing.T) {
	if err := newServiceScenario(t).
		withCreateError(fmt.Errorf("boom")).
		upsert("test body"); err == nil {
		t.Error("expected error when CreateIssueComment fails")
	}
}

func TestService_UpsertComment_UpdateError(t *testing.T) {
	if err := newServiceScenario(t).
		withComments(&gh.IssueComment{ID: gh.Ptr(int64(7)), Body: gh.Ptr(ci.CommentMarker + " existing")}).
		withUpdateError(fmt.Errorf("boom")).
		upsert("test body"); err == nil {
		t.Error("expected error when UpdateIssueComment fails")
	}
}
