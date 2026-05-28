package jiracli

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/coryb/oreo"
)

// ValidateToken probes /rest/api/2/myself to confirm the configured PAT (or
// api-token) is still accepted before doing real work. Per AES guidance we
// fail closed on revoked/expired tokens rather than retrying in a loop.
//
// Returns nil when validation is skipped (auth method is session/cookie or
// the caller opted out).
func ValidateToken(o *oreo.Client, opts *GlobalOptions) error {
	if !opts.AuthMethodIsToken() {
		return nil
	}
	if opts.Endpoint.Value == "" {
		return nil
	}

	uri := strings.TrimRight(opts.Endpoint.Value, "/") + "/rest/api/2/myself"

	// Use a no-retry, no-postcallback client for this probe so we get a
	// clean 401 if the token is bad — no loops, no double-prompts.
	probe := o.WithRetries(0).WithoutPostCallbacks()

	resp, err := probe.GetJSON(uri)
	if err != nil {
		return fmt.Errorf("token validation request failed: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		return nil
	case http.StatusUnauthorized:
		opts.ClearCachedPass()
		log.Errorf(
			"%s msg=\"token validation failed (401)\" hint=\"replace the PAT or re-authenticate; not retrying\"",
			logContext(opts),
		)
		return fmt.Errorf("authentication failed: token is revoked, expired, or rejected")
	case http.StatusForbidden:
		log.Errorf(
			"%s msg=\"token validation forbidden (403)\" hint=\"token lacks required permissions\"",
			logContext(opts),
		)
		return fmt.Errorf("forbidden: token does not have the required permissions")
	default:
		log.Warningf(
			"%s status=%d msg=\"token validation returned unexpected status; continuing\"",
			logContext(opts),
			resp.StatusCode,
		)
		return nil
	}
}
