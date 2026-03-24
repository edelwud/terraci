package gitlab

import (
	"strings"

	gitlab "gitlab.com/gitlab-org/api/client-go"

	"github.com/edelwud/terraci/pkg/ci"
)

// FindTerraCIComment finds an existing terraci comment in the GitLab notes
func FindTerraCIComment(notes []*gitlab.Note) *gitlab.Note {
	for _, note := range notes {
		if strings.Contains(note.Body, ci.CommentMarker) {
			return note
		}
	}
	return nil
}
