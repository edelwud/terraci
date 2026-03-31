package mr

import (
	gitlab "gitlab.com/gitlab-org/api/client-go"

	"github.com/edelwud/terraci/pkg/ciprovider"
)

// FindTerraCIComment finds an existing terraci comment in GitLab notes.
func FindTerraCIComment(notes []*gitlab.Note) *gitlab.Note {
	for _, note := range notes {
		if ciprovider.HasCommentMarker(note.Body) {
			return note
		}
	}
	return nil
}
