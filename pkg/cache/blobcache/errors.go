package blobcache

import "errors"

var (
	// ErrStoreNotConfigured is returned when a cache operation requires a blob store.
	ErrStoreNotConfigured = errors.New("blob store is not configured")
	// ErrEntryNotFound is returned when a cache entry is missing.
	ErrEntryNotFound = errors.New("cache entry not found")
)
