package tft

import (
	"testing"
	"time"
)

func TestIPRateLimiterAllowsBurstThenBlocksThenRefills(t *testing.T) {
	clock := time.Unix(0, 0)
	l := newIPRateLimiter(1 /* rps */, 3 /* burst */, time.Minute)
	l.now = func() time.Time { return clock }

	// Burst of 3 should be allowed immediately.
	for i := 0; i < 3; i++ {
		if !l.allow("1.1.1.1") {
			t.Fatalf("request %d within burst should be allowed", i+1)
		}
	}
	// 4th request with no time elapsed is blocked.
	if l.allow("1.1.1.1") {
		t.Fatal("request beyond burst should be blocked")
	}

	// After 1 second, 1 token refills (rps=1) → one more allowed, then blocked.
	clock = clock.Add(time.Second)
	if !l.allow("1.1.1.1") {
		t.Fatal("request after refill should be allowed")
	}
	if l.allow("1.1.1.1") {
		t.Fatal("second request after single refill should be blocked")
	}
}

func TestIPRateLimiterIsolatesIPs(t *testing.T) {
	clock := time.Unix(0, 0)
	l := newIPRateLimiter(1, 1, time.Minute)
	l.now = func() time.Time { return clock }

	if !l.allow("1.1.1.1") {
		t.Fatal("first IP should be allowed")
	}
	// Different IP has its own bucket, must still be allowed.
	if !l.allow("2.2.2.2") {
		t.Fatal("second IP should have an independent bucket")
	}
	// First IP is now out of tokens.
	if l.allow("1.1.1.1") {
		t.Fatal("first IP should be blocked after spending its burst")
	}
}

func TestIPRateLimiterSweepEvictsIdle(t *testing.T) {
	clock := time.Unix(0, 0)
	l := newIPRateLimiter(1, 1, time.Minute)
	l.now = func() time.Time { return clock }

	l.allow("1.1.1.1")
	if len(l.buckets) != 1 {
		t.Fatalf("expected 1 bucket, got %d", len(l.buckets))
	}

	clock = clock.Add(2 * time.Minute) // past TTL
	l.sweep()
	if len(l.buckets) != 0 {
		t.Fatalf("expected idle bucket to be evicted, got %d", len(l.buckets))
	}
}

func TestRateLimitMiddlewareDisabledWhenNonPositive(t *testing.T) {
	// rps <= 0 must produce a passthrough middleware (non-nil, no limiting).
	if RateLimitMiddleware(0, 5) == nil {
		t.Fatal("disabled limiter should still return a non-nil passthrough handler")
	}
}
