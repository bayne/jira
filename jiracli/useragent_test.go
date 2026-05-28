package jiracli

import (
	"strings"
	"testing"

	"github.com/coryb/figtree"
)

func TestBuildUserAgent_FullySpecified(t *testing.T) {
	opts := &GlobalOptions{
		AppName:      figtree.StringOption{Value: "IssueExporter"},
		Contact:      figtree.StringOption{Value: "proj-foo@disney.com"},
		Runtime:      figtree.StringOption{Value: "aws-lambda"},
		AwsAccountID: figtree.StringOption{Value: "123456789012"},
		AwsRegion:    figtree.StringOption{Value: "us-west-2"},
		Environment:  figtree.StringOption{Value: "nonprod"},
	}
	ua := BuildUserAgent(opts)
	for _, want := range []string{
		"IssueExporter/",
		"(proj-foo@disney.com)",
		"aws-lambda",
		"acct 123456789012",
		"us-west-2",
		"nonprod",
	} {
		if !strings.Contains(ua, want) {
			t.Errorf("expected %q in UA, got %q", want, ua)
		}
	}
}

func TestBuildUserAgent_DefaultsAreNotGeneric(t *testing.T) {
	opts := &GlobalOptions{}
	ua := BuildUserAgent(opts)
	for _, banned := range []string{"curl/", "python-requests"} {
		if strings.Contains(ua, banned) {
			t.Errorf("UA looks generic: %q", ua)
		}
	}
	if !strings.Contains(ua, "go-jira/") {
		t.Errorf("expected go-jira/<version> prefix, got %q", ua)
	}
	if !strings.Contains(ua, "(") || !strings.Contains(ua, ")") {
		t.Errorf("expected contact in parens, got %q", ua)
	}
}
