package blobcache

import (
	"maps"
	"time"
)

// Entry describes one cached blob entry together with TTL-derived timing.
type Entry struct {
	Key       string
	Meta      Meta
	Age       time.Duration
	ExpiresIn time.Duration
}

func cloneMeta(meta Meta) Meta {
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
