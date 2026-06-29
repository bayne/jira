package jiracli

import (
	"context"
	"testing"
	"time"
)

// t0 is an arbitrary fixed base time; tests advance from it deterministically
// so token accounting can be verified without real sleeps.
var t0 = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

func TestTokenBucket_BurstThenThrottle(t *testing.T) {
	// rate 10/s, burst 5: the first 5 requests pass immediately, the 6th must
	// wait ~1/10s for the next token.
	tb := newTokenBucket(10, 5)
	for i := 0; i < 5; i++ {
		if d, ok := tb.take(t0); !ok {
			t.Fatalf("request %d should pass within burst, got wait %v", i+1, d)
		}
	}
	d, ok := tb.take(t0)
	if ok {
		t.Fatal("6th request should be throttled once the burst is spent")
	}
	if want := 100 * time.Millisecond; d != want {
		t.Errorf("want wait %v for empty bucket at 10/s, got %v", want, d)
	}
}

func TestTokenBucket_RefillsOverTime(t *testing.T) {
	tb := newTokenBucket(10, 5)
	for i := 0; i < 5; i++ {
		tb.take(t0) // drain the burst
	}
	// 50ms later, half a token (10/s * 0.05s) has accrued: still not enough.
	if d, ok := tb.take(t0.Add(50 * time.Millisecond)); ok {
		t.Fatalf("0.5 tokens should not satisfy a request, got ok with wait %v", d)
	} else if want := 50 * time.Millisecond; d != want {
		t.Errorf("want %v wait for remaining half token, got %v", want, d)
	}
	// A full second after draining refills well past one token.
	if _, ok := tb.take(t0.Add(time.Second)); !ok {
		t.Error("a request one second after draining should pass")
	}
}

func TestTokenBucket_RefillCapsAtBurst(t *testing.T) {
	// Idle far longer than it takes to fill the bucket; capacity must not grow
	// beyond `max`, so only `burst` requests pass back-to-back afterward.
	tb := newTokenBucket(10, 3)
	for i := 0; i < 3; i++ {
		tb.take(t0)
	}
	later := t0.Add(time.Hour)
	for i := 0; i < 3; i++ {
		if _, ok := tb.take(later); !ok {
			t.Fatalf("request %d should pass from the refilled (capped) bucket", i+1)
		}
	}
	if _, ok := tb.take(later); ok {
		t.Error("bucket refilled past its burst cap; 4th back-to-back request should throttle")
	}
}

func TestTokenBucket_FirstRequestImmediateForSubUnitRate(t *testing.T) {
	// rate < 1/s with no explicit burst: minimum burst of 1 means the first
	// request still goes out without waiting.
	tb := newTokenBucket(0.5, 0)
	if d, ok := tb.take(t0); !ok {
		t.Errorf("first request must pass immediately even at 0.5/s, got wait %v", d)
	}
}

func TestTokenBucket_WaitRespectsContextCancellation(t *testing.T) {
	tb := newTokenBucket(0.001, 1) // ~1000s per token after the first
	if _, ok := tb.take(time.Now()); !ok {
		t.Fatal("first take should consume the single starting token")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already cancelled: wait must return promptly, not block ~1000s
	done := make(chan error, 1)
	go func() { done <- tb.wait(ctx) }()
	select {
	case err := <-done:
		if err != context.Canceled {
			t.Errorf("want context.Canceled, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("wait did not honor cancelled context")
	}
}
