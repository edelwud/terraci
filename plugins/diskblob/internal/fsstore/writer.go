package fsstore

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/edelwud/terraci/pkg/plugin"
)

// ObjectWriter persists blob data and metadata.
type ObjectWriter interface {
	Write(ctx context.Context, paths ObjectPaths, r io.Reader, opts plugin.PutBlobOptions) (plugin.BlobMeta, error)
}

// FileObjectWriter writes blobs to the local filesystem.
type FileObjectWriter struct {
	metadata MetadataCodec
	clock    Clock
}

// NewFileObjectWriter constructs the default filesystem object writer.
func NewFileObjectWriter(metadata MetadataCodec, clock Clock) FileObjectWriter {
	return FileObjectWriter{metadata: metadata, clock: clock}
}

// Write persists blob data and metadata using temp-file + rename.
func (w FileObjectWriter) Write(ctx context.Context, paths ObjectPaths, r io.Reader, opts plugin.PutBlobOptions) (plugin.BlobMeta, error) {
	if err := os.MkdirAll(filepath.Dir(paths.DataPath), 0o755); err != nil {
		return plugin.BlobMeta{}, fmt.Errorf("ensure blob directory: %w", err)
	}

	tmpFile, err := os.CreateTemp(filepath.Dir(paths.DataPath), "blob-*")
	if err != nil {
		return plugin.BlobMeta{}, fmt.Errorf("create blob temp file: %w", err)
	}
	tmpName := tmpFile.Name()
	defer func() {
		_ = tmpFile.Close()
		_ = os.Remove(tmpName)
	}()

	hasher := sha256.New()
	written, err := io.Copy(io.MultiWriter(tmpFile, hasher), r)
	if err != nil {
		return plugin.BlobMeta{}, fmt.Errorf("copy blob data: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return plugin.BlobMeta{}, fmt.Errorf("blob write canceled: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return plugin.BlobMeta{}, fmt.Errorf("close blob temp file: %w", err)
	}
	if err := os.Rename(tmpName, paths.DataPath); err != nil {
		return plugin.BlobMeta{}, fmt.Errorf("rename blob temp file: %w", err)
	}

	meta := plugin.BlobMeta{
		Size:        written,
		UpdatedAt:   w.clock.Now(),
		ExpiresAt:   cloneTimePtr(opts.ExpiresAt),
		ETag:        hex.EncodeToString(hasher.Sum(nil)),
		ContentType: opts.ContentType,
		Metadata:    cloneStringMap(opts.Metadata),
	}

	if err := w.metadata.Write(paths.MetaPath, meta); err != nil {
		return plugin.BlobMeta{}, fmt.Errorf("write metadata: %w", err)
	}

	return meta, nil
}
