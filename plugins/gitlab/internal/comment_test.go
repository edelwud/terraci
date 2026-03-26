package gitlabci

import (
	"testing"

	gitlab "gitlab.com/gitlab-org/api/client-go"

	"github.com/edelwud/terraci/pkg/ci"
)

func TestFindTerraCIComment(t *testing.T) {
	notes := []*gitlab.Note{
		{ID: 1, Body: "Some other comment"},
		{ID: 2, Body: ci.CommentMarker + "\n\n## Terraform Plan"},
		{ID: 3, Body: "Another comment"},
	}

	found := FindTerraCIComment(notes)
	if found == nil {
		t.Fatal("expected to find terraci comment")
	}
	if found.ID != 2 {
		t.Errorf("expected note ID 2, got %d", found.ID)
	}
}

func TestFindTerraCIComment_NotFound(t *testing.T) {
	notes := []*gitlab.Note{
		{ID: 1, Body: "Some comment"},
		{ID: 2, Body: "Another comment"},
	}

	found := FindTerraCIComment(notes)
	if found != nil {
		t.Error("expected nil, found a comment")
	}
}

func TestFindTerraCIComment_NilNotes(t *testing.T) {
	found := FindTerraCIComment(nil)
	if found != nil {
		t.Error("expected nil for nil notes input")
	}
}

func TestFindTerraCIComment_EmptyNotes(t *testing.T) {
	found := FindTerraCIComment([]*gitlab.Note{})
	if found != nil {
		t.Error("expected nil for empty notes input")
	}
}

func TestFindTerraCIComment_FirstMatch(t *testing.T) {
	notes := []*gitlab.Note{
		{ID: 1, Body: ci.CommentMarker + " first"},
		{ID: 2, Body: ci.CommentMarker + " second"},
	}

	found := FindTerraCIComment(notes)
	if found == nil {
		t.Fatal("expected to find comment")
	}
	if found.ID != 1 {
		t.Errorf("expected first match (ID=1), got ID=%d", found.ID)
	}
}
