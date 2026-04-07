package mr

import (
	gitlab "gitlab.com/gitlab-org/api/client-go"

	"github.com/edelwud/terraci/pkg/ci"
)

// FindTerraCIComment finds an existing terraci comment in GitLab notes.
func FindTerraCIComment(notes []*gitlab.Note) *gitlab.Note {
	for _, note := range notes {
		if ci.HasCommentMarker(note.Body) {
			return note
		}
	}
	return nil
}
