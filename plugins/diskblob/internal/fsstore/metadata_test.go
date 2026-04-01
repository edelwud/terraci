package fsstore

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/edelwud/terraci/pkg/plugin"
)

func TestFileMetadataCodecRoundTrip(t *testing.T) {
	t.Parallel()

	rootDir := t.TempDir()
	metaPath := filepath.Join(rootDir, "blob.meta.json")
	expiresAt := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	want := plugin.BlobMeta{
		Size:        42,
		UpdatedAt:   time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC),
		ExpiresAt:   &expiresAt,
		ETag:        "abc123",
		ContentType: "application/json",
		Metadata: map[string]string{
			"kind": "pricing",
		},
	}

	codec := FileMetadataCodec{}
	if err := codec.Write(metaPath, want); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	got, err := codec.Read(metaPath)
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}

	if got.Size != want.Size || got.ETag != want.ETag || got.ContentType != want.ContentType {
		t.Fatalf("Read() basic fields = %+v, want %+v", got, want)
	}
	if got.ExpiresAt == nil || !got.ExpiresAt.Equal(*want.ExpiresAt) {
		t.Fatalf("Read().ExpiresAt = %v, want %v", got.ExpiresAt, want.ExpiresAt)
	}
	if got.Metadata["kind"] != "pricing" {
		t.Fatalf("Read().Metadata = %+v, want pricing marker", got.Metadata)
	}
}
