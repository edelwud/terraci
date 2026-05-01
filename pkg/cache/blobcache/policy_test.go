package blobcache

import (
	"testing"
	"time"
)

type fixedClock struct {
	now time.Time
}

func (c fixedClock) Now() time.Time {
	return c.now
}

func TestPolicyTiming(t *testing.T) {
	now := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	expiresAt := now.Add(30 * time.Minute)
	policy := Policy{
		TTL:   time.Hour,
		Clock: fixedClock{now: now},
	}

	meta := Meta{
		UpdatedAt: now.Add(-10 * time.Minute),
	}
	if got := policy.age(meta); got != 10*time.Minute {
		t.Fatalf("age() = %v, want %v", got, 10*time.Minute)
	}
	if got := policy.expiresIn(meta); got != 50*time.Minute {
		t.Fatalf("expiresIn() = %v, want %v", got, 50*time.Minute)
	}
	if policy.isExpired(meta) {
		t.Fatal("isExpired() = true, want false")
	}

	meta.ExpiresAt = &expiresAt
	if got := policy.expiresIn(meta); got != 30*time.Minute {
		t.Fatalf("expiresIn() with explicit expiry = %v, want %v", got, 30*time.Minute)
	}
	if policy.isExpired(meta) {
		t.Fatal("isExpired() with explicit expiry = true, want false")
	}

	past := now.Add(-time.Minute)
	meta.ExpiresAt = &past
	if !policy.isExpired(meta) {
		t.Fatal("isExpired() with past expiry = false, want true")
	}
}
