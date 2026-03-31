package pr

import (
	"testing"

	gh "github.com/google/go-github/v68/github"

	"github.com/edelwud/terraci/pkg/ci"
)

func TestFindTerraCIComment(t *testing.T) {
	comments := []*gh.IssueComment{
		{ID: gh.Ptr(int64(1)), Body: gh.Ptr("Some other comment")},
		{ID: gh.Ptr(int64(2)), Body: gh.Ptr(ci.CommentMarker + "\n\n## Terraform Plan")},
		{ID: gh.Ptr(int64(3)), Body: gh.Ptr("Another comment")},
	}

	found := FindTerraCIComment(comments)
	if found == nil {
		t.Fatal("expected to find terraci comment")
	}
	if found.GetID() != 2 {
		t.Errorf("expected note ID 2, got %d", found.GetID())
	}
}

func TestFindTerraCIComment_NotFound(t *testing.T) {
	comments := []*gh.IssueComment{
		{ID: gh.Ptr(int64(1)), Body: gh.Ptr("Some comment")},
		{ID: gh.Ptr(int64(2)), Body: gh.Ptr("Another comment")},
	}

	if found := FindTerraCIComment(comments); found != nil {
		t.Error("expected nil, found a comment")
	}
}

func TestFindTerraCIComment_NilComments(t *testing.T) {
	if found := FindTerraCIComment(nil); found != nil {
		t.Error("expected nil for nil comments input")
	}
}

func TestFindTerraCIComment_FirstMatch(t *testing.T) {
	comments := []*gh.IssueComment{
		{ID: gh.Ptr(int64(1)), Body: gh.Ptr(ci.CommentMarker + " first")},
		{ID: gh.Ptr(int64(2)), Body: gh.Ptr(ci.CommentMarker + " second")},
	}

	found := FindTerraCIComment(comments)
	if found == nil {
		t.Fatal("expected to find comment")
	}
	if found.GetID() != 1 {
		t.Errorf("expected first match (ID=1), got ID=%d", found.GetID())
	}
}
