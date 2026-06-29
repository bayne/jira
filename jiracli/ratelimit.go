package jiracli

import (
	"context"
	"sync"
	"time"
)

// tokenBucket is a classic token-bucket rate limiter: the bucket holds up to
// `max` tokens and refills continuously at `rate` tokens per second. Each
// outbound request removes one token before it is sent; if the bucket is empty
// the caller waits until a token accrues. This caps steady-state throughput at
// `rate` req/s while still allowing a short burst of up to `max` requests when
// the bucket has filled during idle time.
//
// It is registered as an oreo pre-callback (see register in cli.go) so it paces
// every command's network egress proactively, complementing the reactive
// exponential backoff in retryCallback that mops up the failures that slip
// through.
type tokenBucket struct {
	mu     sync.Mutex
	tokens float64   // currently available tokens (may be fractional)
	max    float64   // bucket capacity / burst size
	rate   float64   // tokens added per second
	last   time.Time // last time tokens were refilled
}

// newTokenBucket builds a limiter that allows `rate` requests per second with a
// burst capacity of `burst`. The bucket starts full so low-volume callers never
// wait. A burst below one whole token is raised to 1 so the very first request
// always passes immediately even when rate < 1 req/s.
func newTokenBucket(rate, burst float64) *tokenBucket {
	if burst < 1 {
		burst = 1
	}
	return &tokenBucket{tokens: burst, max: burst, rate: rate}
}

// take refills the bucket for the time elapsed since the previous call and then
// attempts to remove one token. If a token was available it returns (0, true)
// after decrementing; otherwise it returns the duration until the next whole
// token will be available and leaves the count unchanged. Time is passed in so
// the token accounting can be unit tested without real sleeps.
func (tb *tokenBucket) take(now time.Time) (time.Duration, bool) {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	// The first call establishes the baseline rather than crediting tokens for
	// the wall-clock time before the limiter existed.
	if tb.last.IsZero() {
		tb.last = now
	}
	if elapsed := now.Sub(tb.last).Seconds(); elapsed > 0 {
		tb.tokens += elapsed * tb.rate
		if tb.tokens > tb.max {
			tb.tokens = tb.max
		}
	}
	tb.last = now

	if tb.tokens >= 1 {
		tb.tokens--
		return 0, true
	}
	// Time for the deficit (1 - tokens) to accrue at `rate` tokens/sec.
	wait := time.Duration((1 - tb.tokens) / tb.rate * float64(time.Second))
	return wait, false
}

// wait blocks until a token is available or the context is cancelled. A
// cancelled context (e.g. a crawl deadline or Ctrl-C) returns its error so the
// caller can abort, matching how retryCallback honors req.Context().
func (tb *tokenBucket) wait(ctx context.Context) error {
	for {
		delay, ok := tb.take(time.Now())
		if ok {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}
}
