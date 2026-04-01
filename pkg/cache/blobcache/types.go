package blobcache

import (
	"maps"
	"time"

	"github.com/edelwud/terraci/pkg/plugin"
)

// PutOptions controls how a cache entry is persisted.
type PutOptions struct {
	ContentType string
	ExpiresAt   *time.Time
	Metadata    map[string]string
}

// Object describes one cached blob object together with TTL-derived timing.
type Object struct {
	Key       string
	Meta      plugin.BlobMeta
	Age       time.Duration
	ExpiresIn time.Duration
}

func cloneBlobMeta(meta plugin.BlobMeta) plugin.BlobMeta {
	meta.Metadata = cloneStringMap(meta.Metadata)
	meta.ExpiresAt = cloneTimePtr(meta.ExpiresAt)
	return meta
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
