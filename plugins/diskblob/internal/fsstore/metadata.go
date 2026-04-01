package fsstore

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"time"

	"github.com/edelwud/terraci/pkg/plugin"
)

type storedMeta struct {
	plugin.BlobMeta
}

// MetadataCodec persists blob metadata sidecars.
type MetadataCodec interface {
	Read(metaPath string) (plugin.BlobMeta, error)
	Write(metaPath string, meta plugin.BlobMeta) error
}

// FileMetadataCodec stores metadata as JSON sidecars.
type FileMetadataCodec struct{}

// Read loads metadata from a sidecar file.
func (FileMetadataCodec) Read(metaPath string) (plugin.BlobMeta, error) {
	data, err := os.ReadFile(metaPath)
	if err != nil {
		return plugin.BlobMeta{}, fmt.Errorf("read metadata: %w", err)
	}

	var meta storedMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return plugin.BlobMeta{}, fmt.Errorf("decode metadata: %w", err)
	}

	meta.Metadata = cloneStringMap(meta.Metadata)
	meta.ExpiresAt = cloneTimePtr(meta.ExpiresAt)
	return meta.BlobMeta, nil
}

// Write stores metadata atomically using a temp file and rename.
func (FileMetadataCodec) Write(metaPath string, meta plugin.BlobMeta) error {
	data, err := json.MarshalIndent(storedMeta{BlobMeta: meta}, "", "  ")
	if err != nil {
		return fmt.Errorf("encode metadata: %w", err)
	}

	if mkdirErr := os.MkdirAll(filepath.Dir(metaPath), 0o755); mkdirErr != nil {
		return fmt.Errorf("ensure metadata directory: %w", mkdirErr)
	}

	tmpFile, err := os.CreateTemp(filepath.Dir(metaPath), "blob-meta-*")
	if err != nil {
		return fmt.Errorf("create metadata temp file: %w", err)
	}
	tmpName := tmpFile.Name()
	defer func() {
		_ = tmpFile.Close()
		_ = os.Remove(tmpName)
	}()

	if _, err := tmpFile.Write(data); err != nil {
		return fmt.Errorf("write metadata temp file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("close metadata temp file: %w", err)
	}
	if err := os.Rename(tmpName, metaPath); err != nil {
		return fmt.Errorf("rename metadata temp file: %w", err)
	}

	return nil
}

func cloneStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}

	out := make(map[string]string, len(in))
	maps.Copy(out, in)

	return out
}

func cloneTimePtr(in *time.Time) *time.Time {
	if in == nil {
		return nil
	}

	value := *in
	return &value
}
