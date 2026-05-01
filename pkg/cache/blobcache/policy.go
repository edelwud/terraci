package blobcache

import "time"

// Policy defines cache timing behavior over blob metadata.
type Policy struct {
	TTL   time.Duration
	Clock Clock
}

func (p Policy) normalized() Policy {
	if p.Clock == nil {
		p.Clock = realClock{}
	}
	return p
}

func (p Policy) expiresIn(meta Meta) time.Duration {
	p = p.normalized()

	if meta.ExpiresAt != nil {
		return meta.ExpiresAt.Sub(p.Clock.Now())
	}
	if p.TTL <= 0 {
		return 0
	}
	return p.TTL - p.age(meta)
}

func (p Policy) age(meta Meta) time.Duration {
	p = p.normalized()
	return p.Clock.Now().Sub(meta.UpdatedAt)
}

func (p Policy) isExpired(meta Meta) bool {
	p = p.normalized()

	if meta.ExpiresAt != nil {
		return p.Clock.Now().After(*meta.ExpiresAt)
	}
	if p.TTL <= 0 {
		return false
	}
	return p.age(meta) > p.TTL
}
