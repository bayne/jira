package jiracli

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/go-jira/jira"
)

// BuildUserAgent produces a User-Agent value following Disney AES REST client
// guidelines.
//
// Format: AppName/semver (contact-dl@example.com) <runtime> [hostname X]
// [acct <id>] [<region>] <environment>
//
// Components fall back to safe defaults so the header is never generic
// (e.g. plain "go-jira/dev"), which would make incidents hard to trace.
func BuildUserAgent(o *GlobalOptions) string {
	name := strings.TrimSpace(o.AppName.Value)
	if name == "" {
		name = "go-jira"
	}
	version := jira.VERSION
	if version == "" {
		version = "0.0.0"
	}

	contact := strings.TrimSpace(o.Contact.Value)
	if contact == "" {
		if u := os.Getenv("USER"); u != "" {
			contact = u + "@unknown"
		} else {
			contact = "unknown@unknown"
		}
	}

	runtimeName := strings.TrimSpace(o.Runtime.Value)
	if runtimeName == "" {
		runtimeName = detectRuntime()
	}

	tokens := []string{}
	if h := strings.TrimSpace(o.Hostname.Value); h != "" {
		tokens = append(tokens, "hostname "+h)
	} else if runtimeName == "local-dev" || runtimeName == "onprem-vm" {
		if host, err := os.Hostname(); err == nil && host != "" {
			tokens = append(tokens, "hostname "+host)
		}
	}
	if a := strings.TrimSpace(o.AwsAccountID.Value); a != "" {
		tokens = append(tokens, "acct "+a)
	}
	if r := strings.TrimSpace(o.AwsRegion.Value); r != "" {
		tokens = append(tokens, r)
	}

	env := strings.TrimSpace(o.Environment.Value)
	if env == "" {
		env = "nonprod"
	}

	head := fmt.Sprintf("%s/%s (%s) %s", name, version, contact, runtimeName)
	parts := []string{head}
	if len(tokens) > 0 {
		parts = append(parts, strings.Join(tokens, " "))
	}
	parts = append(parts, env)
	parts = append(parts, fmt.Sprintf("go/%s/%s", runtime.GOOS, runtime.GOARCH))
	return strings.Join(parts, " ")
}

func detectRuntime() string {
	switch {
	case os.Getenv("AWS_LAMBDA_FUNCTION_NAME") != "":
		return "aws-lambda"
	case os.Getenv("ECS_CONTAINER_METADATA_URI") != "" || os.Getenv("ECS_CONTAINER_METADATA_URI_V4") != "":
		return "aws-ecs"
	case os.Getenv("KUBERNETES_SERVICE_HOST") != "":
		return "kubernetes"
	case os.Getenv("JENKINS_URL") != "" || os.Getenv("BUILD_NUMBER") != "":
		return "jenkins"
	case os.Getenv("CI") != "" || os.Getenv("GITHUB_ACTIONS") != "":
		return "ci"
	default:
		return "local-dev"
	}
}
