package jiracli

import (
	"regexp"
)

// authRedactRE matches Authorization headers regardless of scheme so we can
// strip credential payloads from trace output before it reaches the log.
var authRedactRE = regexp.MustCompile(`(?i)(Authorization:\s*(Bearer|Basic|Token)\s+)\S+`)

// cookieRedactRE strips session/PAT cookie values that some endpoints set.
var cookieRedactRE = regexp.MustCompile(`(?i)((?:Cookie|Set-Cookie):[^\r\n]*?(?:JSESSIONID|atlassian\.xsrf\.token|seraph\.confluence|crowd\.token_key)\s*=\s*)[^\s;]+`)

// queryTokenRE catches credentials accidentally placed in query strings.
var queryTokenRE = regexp.MustCompile(`(?i)([?&](?:token|api_key|access_token|password)=)[^&\s]+`)

// RedactString runs the standard set of secret-scrubbing patterns and
// returns the cleaned text. It is a pure function so it is safe to use on
// both individual log messages and large http dumps.
func RedactString(s string) string {
	s = authRedactRE.ReplaceAllString(s, "${1}[REDACTED]")
	s = cookieRedactRE.ReplaceAllString(s, "${1}[REDACTED]")
	s = queryTokenRE.ReplaceAllString(s, "${1}[REDACTED]")
	return s
}

// RedactBytes is the []byte equivalent of RedactString.
func RedactBytes(b []byte) []byte {
	return []byte(RedactString(string(b)))
}

// RedactArgs scans Printf-style arguments and redacts strings and byte
// slices in place, leaving other values untouched.
func RedactArgs(args []interface{}) []interface{} {
	out := make([]interface{}, len(args))
	for i, a := range args {
		switch x := a.(type) {
		case string:
			out[i] = RedactString(x)
		case []byte:
			out[i] = RedactBytes(x)
		default:
			out[i] = a
		}
	}
	return out
}
