package fsstore

import "time"

// Clock abstracts time for deterministic metadata generation in tests.
type Clock interface {
	Now() time.Time
}

type realClock struct{}

func (realClock) Now() time.Time {
	return time.Now().UTC()
}
