package pr

import (
	gh "github.com/google/go-github/v68/github"

	"github.com/edelwud/terraci/pkg/ci"
)

func FindTerraCIComment(comments []*gh.IssueComment) *gh.IssueComment {
	for _, comment := range comments {
		if comment == nil || comment.Body == nil {
			continue
		}
		if ci.HasCommentMarker(*comment.Body) {
			return comment
		}
	}
	return nil
}
