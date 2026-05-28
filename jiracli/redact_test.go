package jiracli

import (
	"strings"
	"testing"
)

func TestRedactString_AuthorizationBearer(t *testing.T) {
	in := "GET /rest/api/2/myself HTTP/1.1\r\nAuthorization: Bearer eyJhbGciOiJIUzI1NiJ9.abc.def\r\nHost: jira\r\n"
	out := RedactString(in)
	if strings.Contains(out, "eyJhbGciOiJIUzI1NiJ9.abc.def") {
		t.Fatalf("bearer token leaked: %q", out)
	}
	if !strings.Contains(out, "[REDACTED]") {
		t.Fatalf("redaction marker missing: %q", out)
	}
}

func TestRedactString_AuthorizationBasic(t *testing.T) {
	in := "Authorization: Basic dXNlcjpwYXNz"
	out := RedactString(in)
	if strings.Contains(out, "dXNlcjpwYXNz") {
		t.Fatalf("basic credentials leaked: %q", out)
	}
}

func TestRedactString_QueryToken(t *testing.T) {
	in := "GET /api?token=hunter2&id=42 HTTP/1.1"
	out := RedactString(in)
	if strings.Contains(out, "hunter2") {
		t.Fatalf("query token leaked: %q", out)
	}
	if !strings.Contains(out, "id=42") {
		t.Fatalf("redaction stripped non-secret param: %q", out)
	}
}

func TestRedactString_LeavesNonSecretsAlone(t *testing.T) {
	in := "GET /rest/api/2/issue/FOO-1 HTTP/1.1\r\nAccept: application/json\r\nHost: jira\r\n"
	out := RedactString(in)
	if out != in {
		t.Fatalf("unexpected redaction:\nwant: %q\ngot:  %q", in, out)
	}
}

func TestRedactArgs_MixedTypes(t *testing.T) {
	args := []interface{}{"Authorization: Bearer SECRET", 42, []byte("Authorization: Basic abcd")}
	out := RedactArgs(args)
	if s, _ := out[0].(string); strings.Contains(s, "SECRET") {
		t.Fatalf("string secret leaked: %v", out[0])
	}
	if v, _ := out[1].(int); v != 42 {
		t.Fatalf("integer arg mangled: %v", out[1])
	}
	if b, _ := out[2].([]byte); strings.Contains(string(b), "abcd") {
		t.Fatalf("byte secret leaked: %s", b)
	}
}
