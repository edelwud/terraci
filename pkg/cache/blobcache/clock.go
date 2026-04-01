package blobcache

import "time"

// Clock provides time for cache policy calculations.
type Clock interface {
	Now() time.Time
}

type realClock struct{}

func (realClock) Now() time.Time {
	return time.Now()
}
