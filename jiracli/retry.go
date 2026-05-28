package jiracli

import (
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/coryb/oreo"
)

// Retry policy parameters for transient HTTP failures.
// These are package-level so callers can override before commands run.
var (
	MaxRetries       = 4
	InitialBackoff   = 500 * time.Millisecond
	MaxBackoff       = 30 * time.Second
	MaxRetryAfter    = 5 * time.Minute
	retryRandSource  = rand.New(rand.NewSource(time.Now().UnixNano()))
)

// retryableStatus returns true for HTTP status codes that AES guidance
// allows us to retry: 408 (timeout), 429 (rate limited), 504 (gateway
// timeout), and 5xx server errors.
func retryableStatus(code int) bool {
	switch code {
	case http.StatusRequestTimeout,
		http.StatusTooManyRequests,
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		return true
	}
	return false
}

// retryCallback returns an oreo PostRequestCallback that retries transient
// failures with exponential backoff plus jitter, honoring Retry-After when
// present on 429 responses.
//
// getClient returns the current oreo.Client so the callback sees mutations
// applied during command preActions (transport, proxy, etc.) rather than a
// stale snapshot from registration time.
func retryCallback(getClient func() *oreo.Client, opts *GlobalOptions) oreo.PostRequestCallback {
	return func(req *http.Request, resp *http.Response) (*http.Response, error) {
		if resp == nil || !retryableStatus(resp.StatusCode) {
			return resp, nil
		}

		for attempt := 1; attempt <= MaxRetries; attempt++ {
			delay := computeBackoff(attempt, resp)
			log.Warningf(
				"%s status=%d method=%s path=%s msg=\"retrying transient failure\" attempt=%d max=%d delay_ms=%d",
				logContext(opts),
				resp.StatusCode,
				req.Method,
				req.URL.Path,
				attempt,
				MaxRetries,
				delay/time.Millisecond,
			)
			if resp.Body != nil {
				resp.Body.Close()
			}

			select {
			case <-req.Context().Done():
				return resp, req.Context().Err()
			case <-time.After(delay):
			}

			next, err := getClient().Do(req)
			if err != nil {
				return next, err
			}
			resp = next
			if !retryableStatus(resp.StatusCode) {
				return resp, nil
			}
		}
		log.Errorf(
			"%s status=%d method=%s path=%s msg=\"giving up after max retries\" max=%d",
			logContext(opts),
			resp.StatusCode,
			req.Method,
			req.URL.Path,
			MaxRetries,
		)
		return resp, nil
	}
}

// computeBackoff yields a per-attempt wait, honoring Retry-After on 429 and
// applying exponential backoff with +/- 25% jitter otherwise.
func computeBackoff(attempt int, resp *http.Response) time.Duration {
	if resp.StatusCode == http.StatusTooManyRequests {
		if d := parseRetryAfter(resp.Header.Get("Retry-After")); d > 0 {
			if d > MaxRetryAfter {
				return MaxRetryAfter
			}
			return d
		}
	}
	base := InitialBackoff << uint(attempt-1)
	if base > MaxBackoff {
		base = MaxBackoff
	}
	// jitter in [-base/4, +base/4)
	span := int64(base / 2)
	if span <= 0 {
		return base
	}
	j := retryRandSource.Int63n(span) - span/2
	return base + time.Duration(j)
}

func parseRetryAfter(v string) time.Duration {
	if v == "" {
		return 0
	}
	if secs, err := strconv.Atoi(v); err == nil && secs >= 0 {
		return time.Duration(secs) * time.Second
	}
	if t, err := http.ParseTime(v); err == nil {
		if wait := time.Until(t); wait > 0 {
			return wait
		}
	}
	return 0
}
