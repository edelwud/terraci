package ciprovider

import (
	"testing"

	"github.com/edelwud/terraci/pkg/ci"
)

func TestCommentEnabled(t *testing.T) {
	t.Run("nil config defaults true", func(t *testing.T) {
		if !CommentEnabled(nil) {
			t.Fatal("expected nil config to default to true")
		}
	})

	t.Run("nil enabled defaults true", func(t *testing.T) {
		if !CommentEnabled(&MRCommentConfig{}) {
			t.Fatal("expected nil enabled to default to true")
		}
	})

	t.Run("explicit true", func(t *testing.T) {
		enabled := true
		if !CommentEnabled(&MRCommentConfig{Enabled: &enabled}) {
			t.Fatal("expected explicit true")
		}
	})

	t.Run("explicit false", func(t *testing.T) {
		enabled := false
		if CommentEnabled(&MRCommentConfig{Enabled: &enabled}) {
			t.Fatal("expected explicit false")
		}
	})
}

func TestHasCommentMarker(t *testing.T) {
	if !HasCommentMarker("before " + ci.CommentMarker + " after") {
		t.Fatal("expected marker to be detected")
	}
	if HasCommentMarker("plain body") {
		t.Fatal("expected plain body to not match marker")
	}
}
