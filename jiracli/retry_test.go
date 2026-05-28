package jiracli

import (
	"net/http"
	"testing"
	"time"
)

func TestRetryableStatus(t *testing.T) {
	retry := []int{408, 429, 500, 502, 503, 504}
	noRetry := []int{200, 201, 204, 301, 302, 400, 401, 403, 404, 409}
	for _, c := range retry {
		if !retryableStatus(c) {
			t.Errorf("status %d should be retryable", c)
		}
	}
	for _, c := range noRetry {
		if retryableStatus(c) {
			t.Errorf("status %d must NOT be retryable", c)
		}
	}
}

func TestParseRetryAfter_Seconds(t *testing.T) {
	if d := parseRetryAfter("7"); d != 7*time.Second {
		t.Errorf("want 7s, got %v", d)
	}
	if d := parseRetryAfter(""); d != 0 {
		t.Errorf("want 0 for empty, got %v", d)
	}
	if d := parseRetryAfter("not-a-number"); d != 0 {
		t.Errorf("want 0 for garbage, got %v", d)
	}
}

func TestComputeBackoff_HonorsRetryAfterOn429(t *testing.T) {
	resp := &http.Response{
		StatusCode: 429,
		Header:     http.Header{"Retry-After": []string{"3"}},
	}
	d := computeBackoff(1, resp)
	if d != 3*time.Second {
		t.Errorf("want exactly 3s on 429 with Retry-After: 3, got %v", d)
	}
}

func TestComputeBackoff_ExponentialWithJitter(t *testing.T) {
	resp := &http.Response{StatusCode: 500, Header: http.Header{}}
	d1 := computeBackoff(1, resp)
	d2 := computeBackoff(2, resp)
	// attempt 2 base is 2x attempt 1, with up to +/- 25% jitter, so it must
	// be strictly larger than attempt 1's lower jitter bound.
	if d2 < d1/2 {
		t.Errorf("attempt 2 backoff %v unexpectedly smaller than attempt 1 %v", d2, d1)
	}
	if d1 <= 0 {
		t.Errorf("backoff must be positive, got %v", d1)
	}
}

func TestComputeBackoff_CapsAtMaxBackoff(t *testing.T) {
	resp := &http.Response{StatusCode: 500, Header: http.Header{}}
	d := computeBackoff(20, resp)
	if d > MaxBackoff+MaxBackoff/4 {
		t.Errorf("backoff %v exceeded cap %v even with jitter slack", d, MaxBackoff)
	}
}
